package roundservice

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot/app/shared/logging"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
)

// -- Service Functions for DeleteRound Flow --

// ValidateRoundDeleteRequest validates the round delete request.
func (s *RoundService) ValidateRoundDeleteRequest(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundDeleteRequestPayload](msg, s.logger)
	if err != nil {
		return s.publishRoundDeleteError(msg, eventPayload, fmt.Errorf("invalid payload: %w", err))
	}

	if eventPayload.RoundID == 0 { // Check if RoundID is zero
		err := fmt.Errorf("round ID cannot be zero")
		return s.publishRoundDeleteError(msg, eventPayload, err)
	}

	if eventPayload.RequestingUserUserID == "" {
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
		return s.publishRoundDeleteError(msg, roundevents.RoundDeleteRequestPayload{
			RoundID:              0,
			RequestingUserUserID: "",
		}, fmt.Errorf("invalid payload: %w", err))
	}

	// Fetch EventMessageID from DB
	eventMessageID, err := s.RoundDB.GetEventMessageID(ctx, eventPayload.RoundID)
	if err != nil {
		return fmt.Errorf("failed to retrieve EventMessageID for round %d: %w", eventPayload.RoundID, err)
	}

	if err := s.RoundDB.DeleteRound(ctx, eventPayload.RoundID); err != nil {
		return s.publishRoundDeleteError(msg, roundevents.RoundDeleteRequestPayload{
			RoundID:              eventPayload.RoundID,
			RequestingUserUserID: "",
		}, err)
	}

	if err := s.EventBus.CancelScheduledMessage(ctx, eventPayload.RoundID); err != nil {
		s.logger.Error("Failed to cancel scheduled messages", "error", err)
		return s.publishRoundDeleteError(msg, roundevents.RoundDeleteRequestPayload{
			RoundID:              eventPayload.RoundID,
			RequestingUserUserID: "",
		}, err)
	}

	// If publishing `round.deleted` fails, return the error immediately
	if err := s.publishEvent(msg, roundevents.RoundDeleted, roundevents.RoundDeletedPayload{
		RoundID:        eventPayload.RoundID,
		EventMessageID: *eventMessageID,
	}); err != nil {
		logging.LogErrorWithMetadata(ctx, s.logger, msg, "Failed to publish round.deleted event", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("failed to publish round.deleted event: %w", err) // Ensure error is returned
	}

	// Success message should only be logged if everything succeeds
	logging.LogInfoWithMetadata(ctx, s.logger, msg, "Round deleted from database and scheduled messages canceled", map[string]interface{}{"round_id": eventPayload.RoundID})
	return nil
}

// publishRoundDeleteError publishes a round.delete.error event with details.
func (s *RoundService) publishRoundDeleteError(msg *message.Message, input roundevents.RoundDeleteRequestPayload, err error) error {
	payload := roundevents.RoundDeleteErrorPayload{
		RoundDeleteRequest: &input,
		Error:              err.Error(),
	}

	if pubErr := s.publishEvent(msg, roundevents.RoundDeleteError, payload); pubErr != nil {
		s.logger.Error("Failed to publish round.delete.error event", "error", pubErr)
		return fmt.Errorf("failed to publish round.delete.error event: %w", pubErr)
	}

	return err
}
