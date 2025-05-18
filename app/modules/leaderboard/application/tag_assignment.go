package leaderboardservice

import (
	"context"
	"fmt"
	"time"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// TagAssignmentRequested handles the TagAssignmentRequested event.
func (s *LeaderboardService) TagAssignmentRequested(ctx context.Context, payload leaderboardevents.TagAssignmentRequestedPayload) (LeaderboardOperationResult, error) {
	s.logger.InfoContext(ctx, "Tag assignment triggered",
		attr.ExtractCorrelationID(ctx),
		attr.String("user_id", string(payload.UserID)),
		attr.String("tag_number", fmt.Sprintf("%v", *payload.TagNumber)),
		attr.String("source", payload.Source),
		attr.String("update_type", payload.UpdateType),
	)

	return s.serviceWrapper(ctx, "TagAssignmentRequested", func(ctx context.Context) (LeaderboardOperationResult, error) {
		// Get the current active leaderboard for validation
		dbStartTime := time.Now()
		currentLeaderboard, err := s.LeaderboardDB.GetActiveLeaderboard(ctx)
		s.metrics.RecordOperationDuration(ctx, "GetActiveLeaderboard", "LeaderboardService", time.Duration(time.Since(dbStartTime).Seconds()))
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to get active leaderboard", attr.ExtractCorrelationID(ctx), attr.Error(err))
			return LeaderboardOperationResult{
				Failure: &leaderboardevents.TagAssignmentFailedPayload{
					UserID:     payload.UserID,
					TagNumber:  payload.TagNumber,
					Source:     payload.Source,
					UpdateType: payload.UpdateType,
					Reason:     fmt.Sprintf("Failed to get active leaderboard: %v", err),
				},
			}, fmt.Errorf("failed to get active leaderboard: %w", err)
		}

		discordID := sharedtypes.DiscordID(payload.UserID)
		tagNumber := *payload.TagNumber

		// Different validation logic based on update type
		if payload.UpdateType == "update" {
			// For updates, we need to first check if the user exists in the leaderboard
			// If user doesn't exist in current leaderboard, it should be treated as an assign operation
			existingData := currentLeaderboard.FindEntryForUser(discordID)
			s.logger.DebugContext(ctx, "Checking for existing user entry in leaderboard",
				attr.String("searched_user_id", string(discordID)),
				attr.Any("leaderboard_data", currentLeaderboard.LeaderboardData),
			)
			if existingData == nil {
				s.logger.WarnContext(ctx, "Update requested for non-existent user",
					attr.ExtractCorrelationID(ctx),
					attr.String("user_id", string(payload.UserID)),
					attr.String("tag_number", fmt.Sprintf("%v", tagNumber)),
				)
				return LeaderboardOperationResult{
					Failure: &leaderboardevents.TagAssignmentFailedPayload{
						UserID:     payload.UserID,
						TagNumber:  payload.TagNumber,
						Source:     payload.Source,
						UpdateType: payload.UpdateType,
						Reason:     "Cannot update tag for user that doesn't exist in leaderboard",
					},
				}, fmt.Errorf("cannot update tag for non-existent user")
			}

			// For updates, we need to check if the new tag conflicts with any OTHER user
			// but we don't need to check if it conflicts with the same user (since we're updating)
			_, err = s.PrepareTagUpdateForExistingUser(currentLeaderboard, discordID, tagNumber)
		} else {
			// Regular assignment validation (checks for conflicts with any user)
			_, err = s.PrepareTagAssignment(currentLeaderboard, discordID, tagNumber)
		}

		// --- Swap logic addition ---
		if err != nil {
			// Check if this is a swap-needed error
			if swapErr, ok := err.(*TagSwapNeededError); ok {
				s.logger.InfoContext(ctx, "Tag swap needed, triggering swap flow",
					attr.ExtractCorrelationID(ctx),
					attr.String("requestor_id", string(swapErr.RequestorID)),
					attr.String("target_id", string(swapErr.TargetID)),
					attr.String("tag_number", fmt.Sprintf("%v", swapErr.TagNumber)),
				)
				return LeaderboardOperationResult{
					Success: &leaderboardevents.TagSwapRequestedPayload{
						RequestorID: swapErr.RequestorID,
						TargetID:    swapErr.TargetID,
					},
				}, nil
			}

			s.logger.WarnContext(ctx, "Tag assignment validation failed",
				attr.ExtractCorrelationID(ctx),
				attr.String("user_id", string(payload.UserID)),
				attr.String("tag_number", fmt.Sprintf("%v", tagNumber)),
				attr.Error(err),
			)

			var failTagNumber *sharedtypes.TagNumber
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
			}, nil // Note: not returning err here so handler doesn't consider this a handler error
		}
		// --- End swap logic addition ---

		// This method handles creating the new leaderboard record, marking the old one inactive,
		// and generating a new UUID for the UpdateID/AssignmentID of the new record.
		dbStartTime = time.Now()
		newAssignmentID, err := s.LeaderboardDB.AssignTag(
			ctx,
			discordID,
			tagNumber,
			payload.Source,
			payload.UpdateID,
			payload.UserID,
		)
		s.metrics.RecordOperationDuration(ctx, "AssignTag", "LeaderboardService", time.Duration(time.Since(dbStartTime).Seconds()))

		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to assign tag in database", attr.ExtractCorrelationID(ctx), attr.Error(err))
			return LeaderboardOperationResult{
				Failure: &leaderboardevents.TagAssignmentFailedPayload{
					UserID:     payload.UserID,
					TagNumber:  payload.TagNumber,
					Source:     payload.Source,
					UpdateType: payload.UpdateType,
					Reason:     fmt.Sprintf("Database error during tag assignment: %v", err),
				},
			}, fmt.Errorf("failed to assign tag in database: %w", err)
		}

		s.logger.InfoContext(ctx, "Tag assignment successful",
			attr.ExtractCorrelationID(ctx),
			attr.String("user_id", string(payload.UserID)),
			attr.String("tag_number", fmt.Sprintf("%v", *payload.TagNumber)),
			attr.String("assignment_id", newAssignmentID.String()),
		)

		// Return success result with the newly generated AssignmentID and source
		return LeaderboardOperationResult{
			Success: &leaderboardevents.TagAssignedPayload{
				UserID:       payload.UserID,
				TagNumber:    payload.TagNumber,
				AssignmentID: newAssignmentID,
				Source:       payload.Source,
			},
		}, nil
	})
}
