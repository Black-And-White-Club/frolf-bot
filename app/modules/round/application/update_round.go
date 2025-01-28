package roundservice

import (
	"context"
	"fmt"
	"strings"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot/app/shared/logging"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// -- Service Functions for UpdateRound Flow --

// ValidateRoundUpdateRequest validates the round update request.
func (s *RoundService) ValidateRoundUpdateRequest(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundUpdateRequestPayload](msg, s.logger)
	if err != nil {
		return s.publishRoundUpdateError(msg, eventPayload, fmt.Errorf("invalid payload: %w", err))
	}

	var errs []string
	if eventPayload.RoundID == "" {
		errs = append(errs, "round ID cannot be empty")
	}
	if eventPayload.Title == nil && eventPayload.Location == nil && eventPayload.EventType == nil && eventPayload.Date == nil && eventPayload.Time == nil {
		errs = append(errs, "at least one field to update must be provided")
	}

	if len(errs) > 0 {
		errMsg := strings.Join(errs, "; ")
		err := fmt.Errorf("%s", errMsg)
		logging.LogErrorWithMetadata(ctx, s.logger, msg, "Round update input validation failed", map[string]interface{}{
			"errors": errs,
			"error":  err.Error(),
		})
		return s.publishRoundUpdateError(msg, eventPayload, fmt.Errorf("%s", errMsg))
	}

	// If validation passes, publish a "round.update.validated" event
	if err := s.publishEvent(msg, roundevents.RoundUpdateValidated, roundevents.RoundUpdateValidatedPayload{
		RoundUpdateRequestPayload: eventPayload,
	}); err != nil {
		logging.LogErrorWithMetadata(ctx, s.logger, msg, "Failed to publish round.update.validated event", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("failed to publish round.update.validated event: %w", err)
	}

	logging.LogInfoWithMetadata(ctx, s.logger, msg, "Round update input validated", map[string]interface{}{"round_id": eventPayload.RoundID})
	return nil
}

// UpdateRoundEntity updates the round entity with the new values.
func (s *RoundService) UpdateRoundEntity(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundFetchedPayload](msg, s.logger)
	if err != nil {
		return s.publishRoundUpdateError(msg, eventPayload.RoundUpdateRequestPayload, fmt.Errorf("invalid payload: %w", err))
	}

	// Update the round entity fields
	if eventPayload.RoundUpdateRequestPayload.Title != nil {
		eventPayload.Round.Title = *eventPayload.RoundUpdateRequestPayload.Title
	}
	if eventPayload.RoundUpdateRequestPayload.Location != nil {
		eventPayload.Round.Location = *eventPayload.RoundUpdateRequestPayload.Location
	}
	if eventPayload.RoundUpdateRequestPayload.EventType != nil {
		eventPayload.Round.EventType = eventPayload.RoundUpdateRequestPayload.EventType
	}
	if eventPayload.RoundUpdateRequestPayload.Date != nil && eventPayload.RoundUpdateRequestPayload.Time != nil {
		eventPayload.Round.StartTime = time.Date(eventPayload.RoundUpdateRequestPayload.Date.Year(), eventPayload.RoundUpdateRequestPayload.Date.Month(), eventPayload.RoundUpdateRequestPayload.Date.Day(), eventPayload.RoundUpdateRequestPayload.Time.Hour(), eventPayload.RoundUpdateRequestPayload.Time.Minute(), 0, 0, time.UTC)
	}

	// Publish a "round.entity.updated" event
	if err := s.publishEvent(msg, roundevents.RoundEntityUpdated, roundevents.RoundEntityUpdatedPayload{
		Round: eventPayload.Round,
	}); err != nil {
		logging.LogErrorWithMetadata(ctx, s.logger, msg, "Failed to publish round.entity.updated event", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("failed to publish round.entity.updated event: %w", err)
	}

	logging.LogInfoWithMetadata(ctx, s.logger, msg, "Round entity updated", map[string]interface{}{"round_id": eventPayload.Round.ID})
	return nil
}

// StoreRoundUpdate updates the round in the database.
func (s *RoundService) StoreRoundUpdate(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundEntityUpdatedPayload](msg, s.logger)
	if err != nil {
		return s.publishRoundUpdateError(msg, roundevents.RoundUpdateRequestPayload{}, fmt.Errorf("invalid payload: %w", err))
	}

	if err := s.RoundDB.UpdateRound(ctx, eventPayload.Round.ID, &eventPayload.Round); err != nil {
		return s.publishRoundUpdateError(msg, roundevents.RoundUpdateRequestPayload{}, err)
	}

	// Publish "round.updated" event
	if err := s.publishEvent(msg, roundevents.RoundUpdated, roundevents.RoundUpdatedPayload{
		RoundID: eventPayload.Round.ID,
	}); err != nil {
		logging.LogErrorWithMetadata(ctx, s.logger, msg, "Failed to publish round.updated event", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("failed to publish round.updated event: %w", err)
	}

	logging.LogInfoWithMetadata(ctx, s.logger, msg, "Round updated in database", map[string]interface{}{"round_id": eventPayload.Round.ID})
	return nil
}

// PublishRoundUpdated publishes a round.update.success event.
func (s *RoundService) PublishRoundUpdated(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundUpdatedPayload](msg, s.logger)
	if err != nil {
		return s.publishRoundUpdateError(msg, roundevents.RoundUpdateRequestPayload{}, fmt.Errorf("invalid payload: %w", err))
	}

	// Publish the "round.update.success" event
	if err := s.publishEvent(msg, roundevents.RoundUpdateSuccess, roundevents.RoundUpdateSuccessPayload(eventPayload)); err != nil {
		logging.LogErrorWithMetadata(ctx, s.logger, msg, "Failed to publish round.update.success event", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("failed to publish round.update.success event: %w", err)
	}

	logging.LogInfoWithMetadata(ctx, s.logger, msg, "Published round.update.success event", map[string]interface{}{"round_id": eventPayload.RoundID})
	return nil
}

// publishRoundUpdateError publishes a round.update.error event with details.
func (s *RoundService) publishRoundUpdateError(msg *message.Message, input roundevents.RoundUpdateRequestPayload, err error) error {
	payload := roundevents.RoundUpdateErrorPayload{
		CorrelationID:      middleware.MessageCorrelationID(msg),
		RoundUpdateRequest: &input,
		Error:              err.Error(),
	}

	if pubErr := s.publishEvent(msg, roundevents.RoundUpdateError, payload); pubErr != nil {
		logging.LogErrorWithMetadata(context.Background(), s.logger, msg, "Failed to publish round.update.error event", map[string]interface{}{
			"error":          pubErr.Error(),
			"original_error": err.Error(),
		})
		return fmt.Errorf("failed to publish round.update.error event: %w, original error: %w", pubErr, err)
	}

	return err
}
