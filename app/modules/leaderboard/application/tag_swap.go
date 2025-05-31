package leaderboardservice

import (
	"context"
	"time"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// TagSwapRequested handles the TagSwapRequested event.
func (s *LeaderboardService) TagSwapRequested(ctx context.Context, payload leaderboardevents.TagSwapRequestedPayload) (LeaderboardOperationResult, error) {
	s.metrics.RecordTagSwapAttempt(ctx, payload.RequestorID, payload.TargetID)

	s.logger.InfoContext(ctx, "Tag swap triggered",
		attr.ExtractCorrelationID(ctx),
		attr.String("requestor_id", string(payload.RequestorID)),
		attr.String("target_id", string(payload.TargetID)),
	)

	return s.serviceWrapper(ctx, "TagSwapRequested", func(ctx context.Context) (LeaderboardOperationResult, error) {
		// --- Moved this check BEFORE the DB call ---
		if payload.RequestorID == payload.TargetID {
			s.logger.ErrorContext(ctx, "Cannot swap tag with self",
				attr.ExtractCorrelationID(ctx),
				attr.String("requestor_id", string(payload.RequestorID)),
			)
			s.metrics.RecordTagSwapFailure(ctx, payload.RequestorID, payload.TargetID, "cannot swap tag with self")
			return LeaderboardOperationResult{
				Failure: &leaderboardevents.TagSwapFailedPayload{
					RequestorID: payload.RequestorID,
					TargetID:    payload.TargetID,
					Reason:      "cannot swap tag with self",
				},
			}, nil
		}
		// --- End of moved check ---

		startTime := time.Now()
		currentLeaderboard, err := s.LeaderboardDB.GetActiveLeaderboard(ctx) // Now this is called after the self-swap check
		s.metrics.RecordOperationDuration(ctx, "GetActiveLeaderboard", "TagSwapRequested", time.Duration(time.Since(startTime).Seconds()))
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to get active leaderboard",
				attr.ExtractCorrelationID(ctx),
				attr.Error(err),
			)
			s.metrics.RecordTagSwapFailure(ctx, payload.RequestorID, payload.TargetID, err.Error())
			return LeaderboardOperationResult{
				Failure: &leaderboardevents.TagSwapFailedPayload{
					RequestorID: payload.RequestorID,
					TargetID:    payload.TargetID,
					Reason:      err.Error(),
				},
			}, nil
		}
		if currentLeaderboard == nil {
			s.logger.ErrorContext(ctx, "No active leaderboard found",
				attr.ExtractCorrelationID(ctx),
			)
			s.metrics.RecordTagSwapFailure(ctx, payload.RequestorID, payload.TargetID, "no active leaderboard found")
			return LeaderboardOperationResult{
				Failure: &leaderboardevents.TagSwapFailedPayload{
					RequestorID: payload.RequestorID,
					TargetID:    payload.TargetID,
					Reason:      "no active leaderboard found",
				},
			}, nil
		}

		s.logger.InfoContext(ctx, "Active Leaderboard Data",
			attr.ExtractCorrelationID(ctx),
			attr.Any("leaderboard_data", currentLeaderboard.LeaderboardData),
		)

		_, requestorExists := s.FindTagByUserID(ctx, currentLeaderboard, sharedtypes.DiscordID(payload.RequestorID))
		_, targetExists := s.FindTagByUserID(ctx, currentLeaderboard, sharedtypes.DiscordID(payload.TargetID))
		if !requestorExists || !targetExists {
			s.logger.ErrorContext(ctx, "One or both users do not have tags on the leaderboard",
				attr.ExtractCorrelationID(ctx),
				attr.String("requestor_id", string(payload.RequestorID)),
				attr.String("target_id", string(payload.TargetID)),
			)
			s.metrics.RecordTagSwapFailure(ctx, payload.RequestorID, payload.TargetID, "one or both users do not have tags on the leaderboard")
			return LeaderboardOperationResult{
				Failure: &leaderboardevents.TagSwapFailedPayload{
					RequestorID: payload.RequestorID,
					TargetID:    payload.TargetID,
					Reason:      "one or both users do not have tags on the leaderboard",
				},
			}, nil
		}

		startTime = time.Now()
		err = s.LeaderboardDB.SwapTags(ctx, payload.RequestorID, payload.TargetID)
		s.metrics.RecordOperationDuration(ctx, "SwapTags", "TagSwapRequested", time.Duration(time.Since(startTime).Seconds()))
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to swap tags in DB",
				attr.ExtractCorrelationID(ctx),
				attr.Error(err),
			)
			s.metrics.RecordTagSwapFailure(ctx, payload.RequestorID, payload.TargetID, err.Error())
			return LeaderboardOperationResult{
				Failure: &leaderboardevents.TagSwapFailedPayload{
					RequestorID: payload.RequestorID,
					TargetID:    payload.TargetID,
					Reason:      err.Error(),
				},
			}, nil
		}

		s.logger.InfoContext(ctx, "Tags swapped successfully",
			attr.ExtractCorrelationID(ctx),
		)

		s.metrics.RecordTagSwapSuccess(ctx, payload.RequestorID, payload.TargetID)

		return LeaderboardOperationResult{
			Success: &leaderboardevents.TagSwapProcessedPayload{
				RequestorID: payload.RequestorID,
				TargetID:    payload.TargetID,
			},
		}, nil
	})
}
