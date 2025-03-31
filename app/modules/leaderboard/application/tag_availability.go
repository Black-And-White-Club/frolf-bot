package leaderboardservice

import (
	"context"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

// ServiceName is a constant for the leaderboard service name used in metrics and logging
const ServiceName = "leaderboard"

// CheckTagAvailability checks the availability of a tag in the database.
func (s *LeaderboardService) CheckTagAvailability(ctx context.Context, msg *message.Message, payload leaderboardevents.TagAvailabilityCheckRequestedPayload) (*leaderboardevents.TagAvailabilityCheckResultPayload, *leaderboardevents.TagAvailabilityCheckFailedPayload, error) {
	result, err := s.serviceWrapper(msg, "CheckTagAvailability", func() (LeaderboardOperationResult, error) {
		// Get the context from the message after it was updated by the service wrapper
		ctx := msg.Context()

		s.logger.Info("Checking tag availability",
			attr.CorrelationIDFromMsg(msg),
			attr.String("user_id", string(payload.UserID)),
			attr.String("tag_number", payload.TagNumber.String()),
		)

		// Use the TagNumber as its native type for business logic
		isAvailable, err := s.LeaderboardDB.CheckTagAvailability(ctx, *payload.TagNumber)
		if err != nil {
			s.logger.Error("Failed to check tag availability",
				attr.CorrelationIDFromMsg(msg),
				attr.String("user_id", string(payload.UserID)),
				attr.String("tag_number", payload.TagNumber.String()),
				attr.Error(err),
			)

			// Record specific tag check failure metric
			s.metrics.RecordTagAvailabilityCheck(false, *payload.TagNumber, ServiceName)

			return LeaderboardOperationResult{
				Failure: &leaderboardevents.TagAvailabilityCheckFailedPayload{
					UserID:    payload.UserID,
					TagNumber: payload.TagNumber,
					Reason:    "failed to check tag availability",
				},
			}, err
		}

		s.logger.Info("Tag availability check result",
			attr.CorrelationIDFromMsg(msg),
			attr.String("user_id", string(payload.UserID)),
			attr.String("tag_number", payload.TagNumber.String()),
			attr.Bool("is_available", isAvailable),
		)

		// Record specific tag check success metric
		s.metrics.RecordTagAvailabilityCheck(isAvailable, *payload.TagNumber, ServiceName)

		return LeaderboardOperationResult{
			Success: &leaderboardevents.TagAvailabilityCheckResultPayload{
				UserID:    payload.UserID,
				TagNumber: payload.TagNumber,
			},
		}, nil
	})

	if err != nil {
		// Type assertion for the failure case
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

	// Type assertion for the success case
	successPayload, ok := result.Success.(*leaderboardevents.TagAvailabilityCheckResultPayload)
	if !ok {
		successPayload = &leaderboardevents.TagAvailabilityCheckResultPayload{
			UserID:    payload.UserID,
			TagNumber: payload.TagNumber,
		}
	}
	return successPayload, nil, nil
}
