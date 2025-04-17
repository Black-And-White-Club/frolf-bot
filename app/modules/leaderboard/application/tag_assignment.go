package leaderboardservice

import (
	"context"
	"fmt"
	"time"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/google/uuid"
)

// TagAssignmentRequested handles the TagAssignmentRequested event.
func (s *LeaderboardService) TagAssignmentRequested(ctx context.Context, payload leaderboardevents.TagAssignmentRequestedPayload) (LeaderboardOperationResult, error) {
	// Log the operation
	s.logger.InfoContext(ctx, "Tag assignment triggered",
		attr.ExtractCorrelationID(ctx),
		attr.String("user_id", string(payload.UserID)),
		attr.String("requesting_user", string(payload.UserID)),
		attr.String("tag_number", fmt.Sprintf("%v", *payload.TagNumber)),
	)

	return s.serviceWrapper(ctx, "TagAssignmentRequested", func(ctx context.Context) (LeaderboardOperationResult, error) {
		// 1. Get the current active leaderboard for validation
		dbStartTime := time.Now()
		currentLeaderboard, err := s.LeaderboardDB.GetActiveLeaderboard(ctx)
		s.metrics.RecordOperationDuration(ctx, "GetActiveLeaderboard", "LeaderboardService", time.Duration(time.Since(dbStartTime).Seconds()))
		if err != nil {
			return LeaderboardOperationResult{
				Failure: &leaderboardevents.TagAssignmentFailedPayload{
					UserID:     payload.UserID,
					TagNumber:  payload.TagNumber,
					Source:     payload.Source,
					UpdateType: payload.UpdateType,
					Reason:     err.Error(),
				},
			}, err
		}

		// 2. Use the utility function to validate and prepare the tag assignment
		discordID := sharedtypes.DiscordID(payload.UserID)
		tagNumber := *payload.TagNumber

		// Validate the tag assignment first using PrepareTagAssignment
		_, err = s.PrepareTagAssignment(currentLeaderboard, discordID, tagNumber)
		if err != nil {
			var failTagNumber *sharedtypes.TagNumber

			// Only include the tag number if it's valid
			if tagNumber >= 0 {
				failTagNumber = payload.TagNumber
			}

			return LeaderboardOperationResult{
				Failure: &leaderboardevents.TagAssignmentFailedPayload{
					UserID:     payload.UserID,
					TagNumber:  failTagNumber,
					Source:     payload.Source,
					UpdateType: payload.UpdateType,
					Reason:     err.Error(),
				},
			}, err
		}

		// 3. If validation passes, call the AssignTag repository method
		updateID := sharedtypes.RoundID(uuid.Nil) // Use nil UUID to let it be auto-generated

		dbStartTime = time.Now()
		err = s.LeaderboardDB.AssignTag(
			ctx,
			discordID,
			tagNumber,
			leaderboarddb.ServiceUpdateSourceCreateUser, // Default source for new users
			updateID,
		)
		s.metrics.RecordOperationDuration(ctx, "AssignTag", "LeaderboardService", time.Duration(time.Since(dbStartTime).Seconds()))

		if err != nil {
			return LeaderboardOperationResult{
				Failure: &leaderboardevents.TagAssignmentFailedPayload{
					UserID:     payload.UserID,
					TagNumber:  payload.TagNumber,
					Source:     payload.Source,
					UpdateType: payload.UpdateType,
					Reason:     err.Error(),
				},
			}, err
		}

		// Log success and return result
		s.logger.InfoContext(ctx, "Tag assignment successful",
			attr.String("user_id", string(payload.UserID)),
			attr.String("tag_number", fmt.Sprintf("%v", *payload.TagNumber)),
		)

		return LeaderboardOperationResult{
			Success: &leaderboardevents.TagAssignedPayload{
				UserID:    payload.UserID,
				TagNumber: payload.TagNumber,
			},
		}, nil
	})
}
