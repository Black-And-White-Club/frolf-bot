package roundservice

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
)

// getRoundsAndParticipantsToUpdateFromRounds returns ALL participants for rounds that have ANY tag changes
func (s *RoundService) getRoundsAndParticipantsToUpdateFromRounds(ctx context.Context, rounds []*roundtypes.Round, changedTags map[sharedtypes.DiscordID]*sharedtypes.TagNumber) []roundtypes.RoundUpdate {
	updates := make([]roundtypes.RoundUpdate, 0)
	affectedUserIDs := make([]sharedtypes.DiscordID, 0, len(changedTags))

	// Extract user IDs for logging
	for userID := range changedTags {
		affectedUserIDs = append(affectedUserIDs, userID)
	}

	s.logger.InfoContext(ctx, "Processing tag updates for upcoming rounds",
		attr.Int("upcoming_rounds_count", len(rounds)),
		attr.Int("changed_tags_count", len(changedTags)),
		attr.Any("affected_users", affectedUserIDs),
	)

	// Check each round for affected participants
	for _, round := range rounds {
		hasChanges := false
		updatedParticipants := make([]roundtypes.Participant, len(round.Participants))

		// Process ALL participants in the round
		for i, participant := range round.Participants {
			updatedParticipants[i] = participant // Copy the participant

			// Update tag if this user has a tag change
			if newTag, exists := changedTags[participant.UserID]; exists {
				s.logger.InfoContext(ctx, "Updating participant tag",
					attr.RoundID("round_id", round.ID),
					attr.String("user_id", string(participant.UserID)),
					attr.Any("old_tag", participant.TagNumber),
					attr.Any("new_tag", newTag),
				)
				updatedParticipants[i].TagNumber = newTag
				hasChanges = true
			}
		}

		// Only include rounds that actually have changes
		if hasChanges {
			update := roundtypes.RoundUpdate{
				RoundID:        round.ID,
				EventMessageID: round.EventMessageID,
				Participants:   updatedParticipants,
				Round:          round,
			}
			updates = append(updates, update)

			s.logger.InfoContext(ctx, "Round marked for update",
				attr.RoundID("round_id", round.ID),
				attr.String("round_title", string(round.Title)),
				attr.Int("total_participants", len(updatedParticipants)),
				attr.String("event_message_id", round.EventMessageID),
			)
		}
	}

	s.logger.InfoContext(ctx, "Rounds requiring updates determined",
		attr.Int("rounds_to_update", len(updates)),
	)

	return updates
}

// UpdateScheduledRoundsWithNewTags updates ALL upcoming rounds that have affected participants
func (s *RoundService) UpdateScheduledRoundsWithNewTags(ctx context.Context, payload roundevents.ScheduledRoundTagUpdatePayload) (RoundOperationResult, error) {
	result, err := s.serviceWrapper(ctx, "UpdateScheduledRoundsWithNewTags", sharedtypes.RoundID(uuid.Nil), func(ctx context.Context) (RoundOperationResult, error) {
		if len(payload.ChangedTags) == 0 {
			s.logger.InfoContext(ctx, "No tag changes received - operation completed")
			return RoundOperationResult{
				Success: &roundevents.TagsUpdatedForScheduledRoundsPayload{
					UpdatedRounds: []roundevents.RoundUpdateInfo{},
					Summary: roundevents.UpdateSummary{
						TotalRoundsProcessed: 0,
						RoundsUpdated:        0,
						ParticipantsUpdated:  0,
					},
				},
			}, nil
		}

		// Get all upcoming rounds first
		allUpcomingRounds, err := s.RoundDB.GetUpcomingRounds(ctx)
		if err != nil {
			return RoundOperationResult{
				Failure: &roundevents.RoundUpdateErrorPayload{
					Error: fmt.Sprintf("failed to get upcoming rounds: %v", err),
				},
			}, nil
		}
		totalUpcomingRounds := len(allUpcomingRounds)

		// Get rounds that need updates
		updates := s.getRoundsAndParticipantsToUpdateFromRounds(ctx, allUpcomingRounds, payload.ChangedTags)

		if len(updates) == 0 {
			s.logger.InfoContext(ctx, "No upcoming rounds have affected participants",
				attr.Int("total_upcoming_rounds", totalUpcomingRounds),
				attr.Int("changed_tags_count", len(payload.ChangedTags)),
			)
			return RoundOperationResult{
				Success: &roundevents.TagsUpdatedForScheduledRoundsPayload{
					UpdatedRounds: []roundevents.RoundUpdateInfo{},
					Summary: roundevents.UpdateSummary{
						TotalRoundsProcessed: totalUpcomingRounds,
						RoundsUpdated:        0,
						ParticipantsUpdated:  0,
					},
				},
			}, nil
		}

		// Perform database updates efficiently
		if err := s.RoundDB.UpdateRoundsAndParticipants(ctx, updates); err != nil {
			s.logger.ErrorContext(ctx, "Failed to update rounds in database", attr.Error(err))
			return RoundOperationResult{
				Failure: &roundevents.RoundUpdateErrorPayload{
					Error: fmt.Sprintf("database update failed: %v", err),
				},
			}, nil
		}

		// Build response payload with detailed information for Discord
		updatedRounds := make([]roundevents.RoundUpdateInfo, len(updates))
		totalParticipantsUpdated := 0

		for i, update := range updates {
			// Count participants that actually had tag changes
			participantsWithChanges := 0
			for _, participant := range update.Participants {
				if _, hasChange := payload.ChangedTags[participant.UserID]; hasChange {
					participantsWithChanges++
				}
			}
			totalParticipantsUpdated += participantsWithChanges

			updatedRounds[i] = roundevents.RoundUpdateInfo{
				RoundID:             update.RoundID,
				EventMessageID:      update.EventMessageID,
				Title:               update.Round.Title,
				StartTime:           update.Round.StartTime,
				Location:            update.Round.Location,
				UpdatedParticipants: update.Participants, // ALL participants (with updates applied)
				ParticipantsChanged: participantsWithChanges,
			}
		}

		successPayload := &roundevents.TagsUpdatedForScheduledRoundsPayload{
			UpdatedRounds: updatedRounds,
			Summary: roundevents.UpdateSummary{
				TotalRoundsProcessed: totalUpcomingRounds,
				RoundsUpdated:        len(updates),
				ParticipantsUpdated:  totalParticipantsUpdated,
			},
		}

		s.logger.InfoContext(ctx, "Successfully updated scheduled rounds with new tags",
			attr.Int("total_upcoming_rounds", totalUpcomingRounds),
			attr.Int("rounds_updated", len(updates)),
			attr.Int("participants_updated", totalParticipantsUpdated),
		)

		return RoundOperationResult{Success: successPayload}, nil
	})

	return result, err
}
