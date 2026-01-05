package leaderboardservice

import (
	"context"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// CheckTagAvailability checks the availability of a tag in the database.
func (s *LeaderboardService) CheckTagAvailability(ctx context.Context, guildID sharedtypes.GuildID, payload leaderboardevents.TagAvailabilityCheckRequestedPayloadV1) (*leaderboardevents.TagAvailabilityCheckResultPayloadV1, *leaderboardevents.TagAvailabilityCheckFailedPayloadV1, error) {
	result, err := s.serviceWrapper(ctx, "CheckTagAvailability", func(ctx context.Context) (LeaderboardOperationResult, error) {
		s.logger.InfoContext(ctx, "Checking tag availability",
			attr.ExtractCorrelationID(ctx),
			attr.String("user_id", string(payload.UserID)),
			attr.String("tag_number", payload.TagNumber.String()),
		)

		availabilityResult, err := s.LeaderboardDB.CheckTagAvailability(ctx, guildID, payload.UserID, *payload.TagNumber)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to check tag availability",
				attr.ExtractCorrelationID(ctx),
				attr.String("user_id", string(payload.UserID)),
				attr.String("tag_number", payload.TagNumber.String()),
				attr.Error(err),
			)

			s.metrics.RecordTagAvailabilityCheck(ctx, false, *payload.TagNumber, leaderboardevents.LeaderboardStreamName)

			return LeaderboardOperationResult{
				Failure: &leaderboardevents.TagAvailabilityCheckFailedPayloadV1{
					GuildID:   guildID,
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
			attr.Bool("is_available", availabilityResult.Available),
			attr.String("reason", availabilityResult.Reason),
		)

		s.metrics.RecordTagAvailabilityCheck(ctx, availabilityResult.Available, *payload.TagNumber, leaderboardevents.LeaderboardStreamName)

		return LeaderboardOperationResult{
			Success: &leaderboardevents.TagAvailabilityCheckResultPayloadV1{
				GuildID:   guildID,
				UserID:    payload.UserID,
				TagNumber: payload.TagNumber,
				Available: availabilityResult.Available,
				Reason:    availabilityResult.Reason,
			},
		}, nil
	})
	if err != nil {
		failurePayload, ok := result.Failure.(*leaderboardevents.TagAvailabilityCheckFailedPayloadV1)
		if !ok {
			failurePayload = &leaderboardevents.TagAvailabilityCheckFailedPayloadV1{
				GuildID:   guildID,
				UserID:    payload.UserID,
				TagNumber: payload.TagNumber,
				Reason:    "unexpected error format",
			}
		}
		return nil, failurePayload, err
	}

	successPayload, ok := result.Success.(*leaderboardevents.TagAvailabilityCheckResultPayloadV1)
	if !ok {
		successPayload = &leaderboardevents.TagAvailabilityCheckResultPayloadV1{
			GuildID:   guildID,
			UserID:    payload.UserID,
			TagNumber: payload.TagNumber,
			Available: false,
		}
	}
	return successPayload, nil, nil
}
