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
	s.metrics.RecordTagSwapAttempt(payload.RequestorID, payload.TargetID)

	s.logger.Info("Tag swap triggered",
		attr.ExtractCorrelationID(ctx),
		attr.String("requestor_id", string(payload.RequestorID)),
		attr.String("target_id", string(payload.TargetID)),
	)

	return s.serviceWrapper(ctx, "TagSwapRequested", func() (LeaderboardOperationResult, error) {
		startTime := time.Now()
		currentLeaderboard, err := s.LeaderboardDB.GetActiveLeaderboard(ctx)
		s.metrics.RecordOperationDuration("GetActiveLeaderboard", "TagSwapRequested", time.Since(startTime).Seconds())
		if err != nil {
			s.logger.Error("Failed to get active leaderboard",
				attr.ExtractCorrelationID(ctx),
				attr.Error(err),
			)
			s.metrics.RecordTagSwapFailure(payload.RequestorID, payload.TargetID, err.Error())
			return LeaderboardOperationResult{
				Failure: &leaderboardevents.TagSwapFailedPayload{
					RequestorID: payload.RequestorID,
					TargetID:    payload.TargetID,
					Reason:      err.Error(),
				},
			}, nil
		}
		if currentLeaderboard == nil {
			s.logger.Error("No active leaderboard found",
				attr.ExtractCorrelationID(ctx),
			)
			s.metrics.RecordTagSwapFailure(payload.RequestorID, payload.TargetID, "no active leaderboard found")
			return LeaderboardOperationResult{
				Failure: &leaderboardevents.TagSwapFailedPayload{
					RequestorID: payload.RequestorID,
					TargetID:    payload.TargetID,
					Reason:      "no active leaderboard found",
				},
			}, nil
		}

		s.logger.Info("Active Leaderboard Data",
			attr.ExtractCorrelationID(ctx),
			attr.Any("leaderboard_data", currentLeaderboard.LeaderboardData),
		)

		_, requestorExists := s.FindTagByUserID(currentLeaderboard, sharedtypes.DiscordID(payload.RequestorID))
		_, targetExists := s.FindTagByUserID(currentLeaderboard, sharedtypes.DiscordID(payload.TargetID))
		if !requestorExists || !targetExists {
			s.logger.Error("One or both users do not have tags on the leaderboard",
				attr.ExtractCorrelationID(ctx),
				attr.String("requestor_id", string(payload.RequestorID)),
				attr.String("target_id", string(payload.TargetID)),
			)
			s.metrics.RecordTagSwapFailure(payload.RequestorID, payload.TargetID, "one or both users do not have tags on the leaderboard")
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
		s.metrics.RecordOperationDuration("SwapTags", "TagSwapRequested", time.Since(startTime).Seconds())
		if err != nil {
			s.logger.Error("Failed to swap tags in DB",
				attr.ExtractCorrelationID(ctx),
				attr.Error(err),
			)
			s.metrics.RecordTagSwapFailure(payload.RequestorID, payload.TargetID, err.Error())
			return LeaderboardOperationResult{
				Failure: &leaderboardevents.TagSwapFailedPayload{
					RequestorID: payload.RequestorID,
					TargetID:    payload.TargetID,
					Reason:      err.Error(),
				},
			}, nil
		}

		s.logger.Info("Tags swapped successfully",
			attr.ExtractCorrelationID(ctx),
		)

		s.metrics.RecordTagSwapSuccess(payload.RequestorID, payload.TargetID)

		return LeaderboardOperationResult{
			Success: &leaderboardevents.TagSwapProcessedPayload{
				RequestorID: payload.RequestorID,
				TargetID:    payload.TargetID,
			},
		}, nil
	})
}
