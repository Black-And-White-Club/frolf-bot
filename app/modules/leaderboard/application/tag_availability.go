package leaderboardservice

import (
	"context"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
)

// CheckTagAvailability checks the availability of a tag in the database.
func (s *LeaderboardService) CheckTagAvailability(ctx context.Context, payload leaderboardevents.TagAvailabilityCheckRequestedPayload) (*leaderboardevents.TagAvailabilityCheckResultPayload, *leaderboardevents.TagAvailabilityCheckFailedPayload, error) {
	result, err := s.serviceWrapper(ctx, "CheckTagAvailability", func() (LeaderboardOperationResult, error) {
		s.logger.InfoContext(ctx, "Checking tag availability",
			attr.ExtractCorrelationID(ctx),
			attr.String("user_id", string(payload.UserID)),
			attr.String("tag_number", payload.TagNumber.String()),
		)

		available, err := s.LeaderboardDB.CheckTagAvailability(ctx, *payload.TagNumber)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to check tag availability",
				attr.ExtractCorrelationID(ctx),
				attr.String("user_id", string(payload.UserID)),
				attr.String("tag_number", payload.TagNumber.String()),
				attr.Error(err),
			)

			s.metrics.RecordTagAvailabilityCheck(ctx, false, *payload.TagNumber, leaderboardevents.LeaderboardStreamName)

			return LeaderboardOperationResult{
				Failure: &leaderboardevents.TagAvailabilityCheckFailedPayload{
					UserID:    payload.UserID,
					TagNumber: payload.TagNumber,
					Reason:    "failed to check tag availability",
				},
			}, err
		}

		s.logger.InfoContext(ctx, "Tag availability check result",
			attr.ExtractCorrelationID(ctx),
			attr.String("user_id", string(payload.UserID)),
			attr.String("tag_number", payload.TagNumber.String()),
			attr.Bool("is_available", available),
		)

		s.metrics.RecordTagAvailabilityCheck(ctx, available, *payload.TagNumber, leaderboardevents.LeaderboardStreamName)

		return LeaderboardOperationResult{
			Success: &leaderboardevents.TagAvailabilityCheckResultPayload{
				UserID:    payload.UserID,
				TagNumber: payload.TagNumber,
				Available: available,
			},
		}, nil
	})
	if err != nil {
		failurePayload, ok := result.Failure.(*leaderboardevents.TagAvailabilityCheckFailedPayload)
		if !ok {
			failurePayload = &leaderboardevents.TagAvailabilityCheckFailedPayload{
				UserID:    payload.UserID,
				TagNumber: payload.TagNumber,
				Reason:    "unexpected error format",
			}
		}
		return nil, failurePayload, err
	}

	successPayload, ok := result.Success.(*leaderboardevents.TagAvailabilityCheckResultPayload)
	if !ok {
		successPayload = &leaderboardevents.TagAvailabilityCheckResultPayload{
			UserID:    payload.UserID,
			TagNumber: payload.TagNumber,
			Available: false,
		}
	}
	return successPayload, nil, nil
}
