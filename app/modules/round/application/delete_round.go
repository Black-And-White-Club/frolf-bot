package roundservice

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/tcr-bot/app/modules/round/domain/events"
	"github.com/Black-And-White-Club/tcr-bot/app/shared/logging"
	"github.com/Black-And-White-Club/tcr-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// -- Service Functions for DeleteRound Flow --

// ValidateRoundDeleteRequest validates the round delete request.
func (s *RoundService) ValidateRoundDeleteRequest(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundDeleteRequestPayload](msg, s.logger)
	if err != nil {
		return s.publishRoundDeleteError(msg, eventPayload, fmt.Errorf("invalid payload: %w", err))
	}

	if eventPayload.RoundID == "" {
		err := fmt.Errorf("round ID cannot be empty")
		return s.publishRoundDeleteError(msg, eventPayload, err)
	}

	if eventPayload.RequestingUserDiscordID == "" {
		err := fmt.Errorf("requesting user's Discord ID cannot be empty")
		return s.publishRoundDeleteError(msg, eventPayload, err)
	}

	// If validation passes, publish a "round.delete.validated" event
	if err := s.publishEvent(msg, roundevents.RoundDeleteValidated, roundevents.RoundDeleteValidatedPayload{
		RoundDeleteRequestPayload: eventPayload,
	}); err != nil {
		logging.LogErrorWithMetadata(ctx, s.logger, msg, "Failed to publish round.delete.validated event", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("failed to publish round.delete.validated event: %w", err)
	}

	logging.LogInfoWithMetadata(ctx, s.logger, msg, "Round delete request validated", map[string]interface{}{"round_id": eventPayload.RoundID})
	return nil
}

// DeleteRound deletes the round from the database.
func (s *RoundService) DeleteRound(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundDeleteAuthorizedPayload](msg, s.logger)
	if err != nil {
		return s.publishRoundDeleteError(msg, roundevents.RoundDeleteRequestPayload{}, fmt.Errorf("invalid payload: %w", err))
	}

	if err := s.RoundDB.DeleteRound(ctx, eventPayload.RoundID); err != nil {
		// Pass RoundID in the error payload
		return s.publishRoundDeleteError(msg, roundevents.RoundDeleteRequestPayload{RoundID: eventPayload.RoundID}, err)
	}

	// Publish "round.deleted" event
	if err := s.publishEvent(msg, roundevents.RoundDeleted, roundevents.RoundDeletedPayload(eventPayload)); err != nil {
		logging.LogErrorWithMetadata(ctx, s.logger, msg, "Failed to publish round.deleted event", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("failed to publish round.deleted event: %w", err)
	}

	logging.LogInfoWithMetadata(ctx, s.logger, msg, "Round deleted from database", map[string]interface{}{"round_id": eventPayload.RoundID})
	return nil
}

// publishRoundDeleteError publishes a round.delete.error event with details.
func (s *RoundService) publishRoundDeleteError(msg *message.Message, input roundevents.RoundDeleteRequestPayload, err error) error {
	payload := roundevents.RoundDeleteErrorPayload{
		CorrelationID:      middleware.MessageCorrelationID(msg),
		RoundDeleteRequest: &input,
		Error:              err.Error(),
	}

	if pubErr := s.publishEvent(msg, roundevents.RoundDeleteError, payload); pubErr != nil {
		logging.LogErrorWithMetadata(context.Background(), s.logger, msg, "Failed to publish round.delete.error event", map[string]interface{}{
			"original_error": err.Error(),
		})
		return fmt.Errorf("failed to publish round.delete.error event: %w, original error: %w", pubErr, err)
	}

	return err
}
