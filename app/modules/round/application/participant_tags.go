package roundservice

import (
	"context"
	"fmt"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
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
				s.logger.InfoContext(ctx, "Syncing participant tag for upcoming round",
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
			count := 0
			for _, p := range round.Participants {
				if _, exists := changedTags[p.UserID]; exists {
					count++
				}
			}
			updates = append(updates, roundtypes.RoundUpdate{
				RoundID:                  round.ID,
				EventMessageID:           round.EventMessageID,
				Participants:             updatedParticipants,
				ParticipantsChangedCount: count,
				Round:                    round,
			})
		}
	}

	return updates
}

// UpdateScheduledRoundsWithNewTags is the primary entry point for syncing the Round module
// with Leaderboard mutations (Manual Swaps, Batch Updates, or Round Results).
func (s *RoundService) UpdateScheduledRoundsWithNewTags(
	ctx context.Context,
	req *roundtypes.UpdateScheduledRoundsWithNewTagsRequest,
) (UpdateScheduledRoundsWithNewTagsResult, error) {
	// Wrap in service logic for tracing/metrics.
	// Use uuid.Nil because this affects multiple potential rounds.
	return withTelemetry[*roundtypes.ScheduledRoundsSyncResult, error](s, ctx, "UpdateScheduledRoundsWithNewTags", sharedtypes.RoundID(uuid.Nil), func(ctx context.Context) (UpdateScheduledRoundsWithNewTagsResult, error) {
		return runInTx[*roundtypes.ScheduledRoundsSyncResult, error](s, ctx, func(ctx context.Context, tx bun.IDB) (UpdateScheduledRoundsWithNewTagsResult, error) {

			if req.GuildID == "" {
				return results.FailureResult[*roundtypes.ScheduledRoundsSyncResult, error](fmt.Errorf("missing guild_id in update request")), nil
			}

			if len(req.ChangedTags) == 0 {
				s.logger.InfoContext(ctx, "No tag changes to sync; skipping round updates")
				return results.SuccessResult[*roundtypes.ScheduledRoundsSyncResult, error](&roundtypes.ScheduledRoundsSyncResult{
					GuildID: req.GuildID,
				}), nil
			}

			// 1. Fetch all upcoming rounds for this guild.
			// This ensures we catch any future events the users are signed up for.
			allUpcomingRounds, err := s.repo.GetUpcomingRounds(ctx, tx, req.GuildID)
			if err != nil {
				s.logger.ErrorContext(ctx, "Failed to retrieve upcoming rounds for tag sync", attr.Error(err))
				return results.FailureResult[*roundtypes.ScheduledRoundsSyncResult, error](fmt.Errorf("failed to get upcoming rounds: %w", err)), nil
			}

			s.logger.InfoContext(ctx, "GetUpcomingRounds result",
				attr.Int("count", len(allUpcomingRounds)),
				attr.String("guild_id", string(req.GuildID)),
			)

			// 2. Determine which rounds need the atomic update
			updates := s.getRoundsAndParticipantsToUpdateFromRounds(ctx, allUpcomingRounds, req.ChangedTags)

			s.logger.InfoContext(ctx, "getRoundsAndParticipantsToUpdateFromRounds result",
				attr.Int("updates_count", len(updates)),
			)

			if len(updates) == 0 {
				s.logger.InfoContext(ctx, "No upcoming rounds affected by tag changes",
					attr.String("guild_id", string(req.GuildID)),
					attr.Int("total_upcoming", len(allUpcomingRounds)))

				return results.SuccessResult[*roundtypes.ScheduledRoundsSyncResult, error](&roundtypes.ScheduledRoundsSyncResult{
					GuildID:      req.GuildID,
					TotalChecked: len(allUpcomingRounds),
				}), nil
			}

			// 3. Batch persist the changes to the Database
			if err := s.repo.UpdateRoundsAndParticipants(ctx, tx, req.GuildID, updates); err != nil {
				s.logger.ErrorContext(ctx, "Database failure during round tag synchronization", attr.Error(err))
				return results.FailureResult[*roundtypes.ScheduledRoundsSyncResult, error](fmt.Errorf("database update failed: %w", err)), nil
			}

			s.logger.InfoContext(ctx, "Successfully synchronized rounds with new leaderboard state",
				attr.Int("rounds_updated", len(updates)))

			return results.SuccessResult[*roundtypes.ScheduledRoundsSyncResult, error](&roundtypes.ScheduledRoundsSyncResult{
				GuildID:      req.GuildID,
				Updates:      updates,
				TotalChecked: len(allUpcomingRounds),
			}), nil
		})
	})
}
