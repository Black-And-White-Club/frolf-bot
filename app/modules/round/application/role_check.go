package roundservice

import (
	"context"
	"fmt"
	"log/slog"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot/app/shared/logging"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
)

// CheckUser Authorization checks if the requesting user is authorized to delete the round.
func (s *RoundService) CheckUserAuthorization(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundToDeleteFetchedPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal RoundToDeleteFetchedPayload: %w", err)
	}

	// Check if the requesting user is the creator of the round
	if eventPayload.Round.CreatedBy == eventPayload.RoundDeleteRequestPayload.RequestingUserUserID {
		// If the user is the creator, publish a "round.delete.authorized" event
		if err := s.publishEvent(msg, roundevents.RoundDeleteAuthorized, roundevents.RoundDeleteAuthorizedPayload{
			RoundID: eventPayload.Round.ID, // Ensure this matches the field name in roundtypes.Round
		}); err != nil {
			logging.LogErrorWithMetadata(ctx, s.logger, msg, "Failed to publish round.delete.authorized event", map[string]interface{}{
				"error": err,
			})
			return fmt.Errorf("failed to publish round.delete.authorized event: %w", err)
		}

		logging.LogInfoWithMetadata(ctx, s.logger, msg, "User  is authorized to delete the round (round creator)", map[string]interface{}{
			"round_id": eventPayload.Round.ID,
			"user_id":  eventPayload.RoundDeleteRequestPayload.RequestingUserUserID,
		})
		return nil
	}

	// If the user is not the creator, publish a "round.user.role.check.request" event
	if err := s.publishEvent(msg, roundevents.RoundUserRoleCheckRequest, roundevents.UserRoleCheckRequestPayload{
		UserID:  eventPayload.RoundDeleteRequestPayload.RequestingUserUserID,
		RoundID: eventPayload.Round.ID,
	}); err != nil {
		logging.LogErrorWithMetadata(ctx, s.logger, msg, "Failed to publish round.user.role.check.request event", map[string]interface{}{
			"error": err,
		})
		return fmt.Errorf("failed to publish round.user.role.check.request event: %w", err)
	}

	logging.LogInfoWithMetadata(ctx, s.logger, msg, "User  role check requested", map[string]interface{}{
		"round_id": eventPayload.Round.ID,
		"user_id":  eventPayload.RoundDeleteRequestPayload.RequestingUserUserID,
	})
	return nil
}

// HandleUser RoleCheckResult handles the result of the user role check.
func (s *RoundService) UserRoleCheckResult(ctx context.Context, msg *message.Message) error {
	correlationID, eventPayload, err := eventutil.UnmarshalPayload[roundevents.UserRoleCheckResultPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal UserRoleCheckResultPayload: %w", err)
	}

	// If the user has the required role, publish a "round.delete.authorized" event
	if eventPayload.HasRole {
		if err := s.publishEvent(msg, roundevents.RoundDeleteAuthorized, roundevents.RoundDeleteAuthorizedPayload{
			RoundID: eventPayload.RoundID,
		}); err != nil {
			logging.LogErrorWithMetadata(ctx, s.logger, msg, "Failed to publish round.delete.authorized event", map[string]interface{}{
				"error": err,
			})
			return fmt.Errorf("failed to publish round.delete.authorized event: %w", err)
		}

		logging.LogInfoWithMetadata(ctx, s.logger, msg, "User  is authorized to delete the round (role check passed)", map[string]interface{}{
			"round_id": eventPayload.RoundID,
			"user_id":  eventPayload.UserID,
		})
		return nil
	}

	// If the user does not have the required role, publish a "round.delete.unauthorized" event
	s.logger.Error("User  is not authorized to delete the round",
		slog.Int64("round_id", int64(eventPayload.RoundID)),
		slog.String("user_id", string(eventPayload.UserID)),
		slog.String("correlation_id", correlationID),
		slog.Any("error", err),
	)
	if err = s.publishEvent(msg, roundevents.RoundDeleteUnauthorized, eventPayload); err != nil {
		logging.LogErrorWithMetadata(ctx, s.logger, msg, "Failed to publish round.delete.unauthorized event", map[string]interface{}{
			"error": err,
		})
		return fmt.Errorf("failed to publish round.delete.unauthorized event: %w", err)
	}

	logging.LogInfoWithMetadata(ctx, s.logger, msg, "Published round.delete.unauthorized event", map[string]interface{}{
		"round_id": eventPayload.RoundID,
		"user_id":  eventPayload.UserID,
	})
	return nil
}
