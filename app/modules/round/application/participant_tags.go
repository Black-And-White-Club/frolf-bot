package roundservice

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/google/uuid"
)

// getRoundsAndParticipantsToUpdateFromRounds returns ALL participants for rounds that have ANY tag changes.
// It uses a map of [UserID]TagNumber to identify which participants need synchronization.
func (s *RoundService) getRoundsAndParticipantsToUpdateFromRounds(
	ctx context.Context,
	rounds []*roundtypes.Round,
	changedTags map[sharedtypes.DiscordID]sharedtypes.TagNumber,
) []roundtypes.RoundUpdate {
	updates := make([]roundtypes.RoundUpdate, 0)

	// Check each upcoming round for affected participants
	for _, round := range rounds {
		hasChanges := false
		updatedParticipants := make([]roundtypes.Participant, len(round.Participants))

		// Process ALL participants in the round to maintain a complete participant set
		for i, participant := range round.Participants {
			updatedParticipants[i] = participant

			// If this specific user has a new tag from the Funnel/Saga, update it
			if newTag, exists := changedTags[participant.UserID]; exists {
				s.logger.DebugContext(ctx, "Syncing participant tag for upcoming round",
					attr.RoundID("round_id", round.ID),
					attr.String("user_id", string(participant.UserID)),
					attr.Any("old_tag", participant.TagNumber),
					attr.Any("new_tag", newTag),
				)
				// take the address of a copy so we assign a *sharedtypes.TagNumber
				t := newTag
				updatedParticipants[i].TagNumber = &t
				hasChanges = true
			}
		}

		// Only include rounds that actually required a state change
		if hasChanges {
			updates = append(updates, roundtypes.RoundUpdate{
				RoundID:        round.ID,
				EventMessageID: round.EventMessageID,
				Participants:   updatedParticipants,
				Round:          round,
			})
		}
	}

	return updates
}

// UpdateScheduledRoundsWithNewTags is the primary entry point for syncing the Round module
// with Leaderboard mutations (Manual Swaps, Batch Updates, or Round Results).
func (s *RoundService) UpdateScheduledRoundsWithNewTags(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	changedTags map[sharedtypes.DiscordID]sharedtypes.TagNumber,
) (results.OperationResult, error) {
	// Wrap in service logic for tracing/metrics.
	// Use uuid.Nil because this affects multiple potential rounds.
	return s.withTelemetry(ctx, "UpdateScheduledRoundsWithNewTags", sharedtypes.RoundID(uuid.Nil), func(ctx context.Context) (results.OperationResult, error) {

		if guildID == "" {
			return results.OperationResult{
				Failure: &roundevents.RoundUpdateErrorPayloadV1{
					Error: "missing guild_id in update request",
				},
			}, nil
		}

		if len(changedTags) == 0 {
			s.logger.InfoContext(ctx, "No tag changes to sync; skipping round updates")
			return results.OperationResult{
				Success: &roundevents.ScheduledRoundsSyncedPayloadV1{
					GuildID: guildID,
					Summary: roundevents.UpdateSummaryV1{GuildID: guildID},
				},
			}, nil
		}

		// 1. Fetch all upcoming rounds for this guild.
		// This ensures we catch any future events the users are signed up for.
		allUpcomingRounds, err := s.repo.GetUpcomingRounds(ctx, guildID)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to retrieve upcoming rounds for tag sync", attr.Error(err))
			return results.OperationResult{
				Failure: &roundevents.RoundUpdateErrorPayloadV1{
					GuildID: guildID,
					Error:   fmt.Sprintf("failed to get upcoming rounds: %v", err),
				},
			}, nil
		}

		// 2. Determine which rounds need the atomic update
		updates := s.getRoundsAndParticipantsToUpdateFromRounds(ctx, allUpcomingRounds, changedTags)

		if len(updates) == 0 {
			s.logger.InfoContext(ctx, "No upcoming rounds affected by tag changes",
				attr.String("guild_id", string(guildID)),
				attr.Int("total_upcoming", len(allUpcomingRounds)))

			return results.OperationResult{
				Success: &roundevents.ScheduledRoundsSyncedPayloadV1{
					GuildID: guildID,
					Summary: roundevents.UpdateSummaryV1{
						GuildID:              guildID,
						TotalRoundsProcessed: len(allUpcomingRounds),
						RoundsUpdated:        0,
					},
				},
			}, nil
		}

		// 3. Batch persist the changes to the Database
		if err := s.repo.UpdateRoundsAndParticipants(ctx, guildID, updates); err != nil {
			s.logger.ErrorContext(ctx, "Database failure during round tag synchronization", attr.Error(err))
			return results.OperationResult{
				Failure: &roundevents.RoundUpdateErrorPayloadV1{
					GuildID: guildID,
					Error:   fmt.Sprintf("database update failed: %v", err),
				},
			}, nil
		}

		// 4. Construct the success event payload
		updatedRounds := make([]roundevents.RoundUpdateInfoV1, len(updates))
		totalParticipantsUpdated := 0

		for i, update := range updates {
			participantsWithChanges := 0
			for _, participant := range update.Participants {
				if _, hasChange := changedTags[participant.UserID]; hasChange {
					participantsWithChanges++
				}
			}
			totalParticipantsUpdated += participantsWithChanges

			updatedRounds[i] = roundevents.RoundUpdateInfoV1{
				GuildID:             guildID,
				RoundID:             update.RoundID,
				EventMessageID:      update.EventMessageID,
				Title:               update.Round.Title,
				StartTime:           update.Round.StartTime,
				Location:            update.Round.Location,
				UpdatedParticipants: update.Participants,
				ParticipantsChanged: participantsWithChanges,
			}
		}

		s.logger.InfoContext(ctx, "Successfully synchronized rounds with new leaderboard state",
			attr.Int("rounds_updated", len(updates)),
			attr.Int("total_users_synced", totalParticipantsUpdated))

		return results.OperationResult{
			Success: &roundevents.ScheduledRoundsSyncedPayloadV1{
				GuildID:       guildID,
				UpdatedRounds: updatedRounds,
				Summary: roundevents.UpdateSummaryV1{
					GuildID:              guildID,
					TotalRoundsProcessed: len(allUpcomingRounds),
					RoundsUpdated:        len(updates),
					ParticipantsUpdated:  totalParticipantsUpdated,
				},
			},
		}, nil
	})
}
