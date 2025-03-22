package roundservice

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/Black-And-White-Club/frolf-bot/app/shared/logging"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
)

// -- Service Functions for UpdateRound Flow --

// ValidateRoundUpdateRequest validates the round update request.
func (s *RoundService) ValidateRoundUpdateRequest(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundUpdateRequestPayload](msg, s.logger)
	if err != nil {
		return s.publishRoundUpdateError(msg, eventPayload, fmt.Errorf("invalid payload: %w", err))
	}

	var errs []string
	if eventPayload.RoundID == 0 {
		errs = append(errs, "round ID cannot be zero")
	}

	if eventPayload.Title == "" && eventPayload.Location == nil && eventPayload.Description == nil && eventPayload.StartTime == nil {
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
	// 1. Unmarshal the payload
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundFetchedPayload](msg, s.logger)
	if err != nil {
		s.logger.Error("Unmarshal failed in UpdateRoundEntity", "error", err)
		return s.publishRoundUpdateError(msg, eventPayload.RoundUpdateRequestPayload, fmt.Errorf("invalid payload: %w", err))
	}

	// 2. Fetch the existing round
	existingRound, err := s.RoundDB.GetRound(ctx, eventPayload.Round.ID)
	if err != nil {
		s.logger.Error("Failed to fetch round", "round_id", eventPayload.Round.ID, "error", err)
		return s.publishRoundUpdateError(msg, eventPayload.RoundUpdateRequestPayload, fmt.Errorf("failed to fetch existing round: %w", err))
	}

	// 3. Apply updates
	if eventPayload.RoundUpdateRequestPayload.Title != "" {
		existingRound.Title = eventPayload.RoundUpdateRequestPayload.Title
	}
	if eventPayload.RoundUpdateRequestPayload.Description != nil {
		existingRound.Description = eventPayload.RoundUpdateRequestPayload.Description
	}
	if eventPayload.RoundUpdateRequestPayload.Location != nil {
		existingRound.Location = eventPayload.RoundUpdateRequestPayload.Location
	}
	if eventPayload.RoundUpdateRequestPayload.StartTime != nil {
		startTime := roundtypes.StartTime(*eventPayload.RoundUpdateRequestPayload.StartTime)
		existingRound.StartTime = &startTime
	}

	// 4. Update the round in the database
	if err = s.RoundDB.UpdateRound(ctx, existingRound.ID, existingRound); err != nil {
		s.logger.Error("Failed to update round entity", "round_id", existingRound.ID, "error", err)
		return s.publishRoundUpdateError(msg, eventPayload.RoundUpdateRequestPayload, fmt.Errorf("failed to update round entity: %w", err))
	}

	// 5. Successfully updated round → Publish "round.updated" event
	if err = s.publishEvent(msg, roundevents.RoundUpdated, roundevents.RoundEntityUpdatedPayload{
		Round: roundtypes.Round{
			ID:           existingRound.ID, // Assuming existingRound.ID is of type roundtypes.ID
			Title:        existingRound.Title,
			Description:  existingRound.Description,
			Location:     existingRound.Location,
			EventType:    existingRound.EventType,
			StartTime:    existingRound.StartTime,
			Finalized:    existingRound.Finalized,
			CreatedBy:    existingRound.CreatedBy,
			State:        existingRound.State,
			Participants: existingRound.Participants,
		},
	}); err != nil {
		s.logger.Error("Failed to publish round.updated event", "round_id", existingRound.ID, "error", err)
		return fmt.Errorf("failed to publish round.updated event: %w", err)
	}

	s.logger.Info("Round successfully updated", "round_id", existingRound.ID)
	return nil
}

// StoreRoundUpdate updates the round in the database.
func (s *RoundService) StoreRoundUpdate(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundEntityUpdatedPayload](msg, s.logger)
	if err != nil {
		return s.publishRoundUpdateError(msg, roundevents.RoundUpdateRequestPayload{}, fmt.Errorf("invalid payload: %w", err))
	}

	// Use eventPayload.Round directly if it is already of the correct type
	dbRound := eventPayload.Round // Assuming eventPayload.Round is of type roundtypes.Round

	// Pass the address of dbRound to UpdateRound
	if err := s.RoundDB.UpdateRound(ctx, dbRound.ID, &dbRound); err != nil { // Take the address of dbRound
		return s.publishRoundUpdateError(msg, roundevents.RoundUpdateRequestPayload{}, err)
	}

	// Publish "round.updated" event
	if err := s.publishEvent(msg, roundevents.RoundUpdated, roundevents.RoundStateUpdatedPayload{
		RoundID: eventPayload.Round.ID,
	}); err != nil {
		logging.LogErrorWithMetadata(ctx, s.logger, msg, "Failed to publish round.updated event", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("failed to publish round.updated event: %w", err)
	}

	logging.LogInfoWithMetadata(ctx, s.logger, msg, "Round updated in database", map[string]interface{}{"round_id": eventPayload.Round.ID})
	return nil
}

// publishRoundUpdateError publishes a round.update.error event with details.
func (s *RoundService) publishRoundUpdateError(msg *message.Message, input roundevents.RoundUpdateRequestPayload, err error) error {
	payload := roundevents.RoundUpdateErrorPayload{
		RoundUpdateRequest: &input,
		Error:              err.Error(),
	}

	if pubErr := s.publishEvent(msg, roundevents.RoundUpdateError, payload); pubErr != nil {
		return fmt.Errorf("failed to publish round.update.error event: %w, original error: %w", pubErr, err)
	}

	return err
}

// UpdateScheduledRoundEvents updates the scheduled events for a round.
func (s *RoundService) UpdateScheduledRoundEvents(ctx context.Context, msg *message.Message) error {
	// Step 1: Unmarshal the payload
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundScheduleUpdatePayload](msg, s.logger)
	if err != nil {
		s.logger.Error("Failed to unmarshal message payload",
			slog.String("message_uuid", msg.UUID),
			slog.Any("error", err),
		)

		errorPayload := roundevents.RoundUpdateErrorPayload{
			RoundUpdateRequest: nil,
			Error:              fmt.Sprintf("failed to unmarshal payload: %v", err),
		}

		if err := s.publishEvent(msg, roundevents.RoundUpdateError, errorPayload); err != nil {
			return fmt.Errorf("error during error reporting: %w", err)
		}
		return fmt.Errorf("invalid payload: %w", err)
	}

	roundID := eventPayload.RoundID
	s.logger.Info("Processing scheduled round update",
		slog.String("round_id", fmt.Sprintf("%d", roundID)),
	)

	// Step 2: Attempt to cancel existing scheduled events
	if err := s.EventBus.CancelScheduledMessage(ctx, roundID); err != nil {
		s.logger.Warn("Failed to cancel existing scheduled events, proceeding anyway",
			slog.String("error", err.Error()),
			slog.String("round_id", fmt.Sprintf("%d", roundID)),
		)
	}

	// Step 3: Fetch the complete round information from the database
	round, fetchErr := s.RoundDB.GetRound(ctx, roundID)
	if fetchErr != nil {
		s.logger.Error("Failed to fetch round for rescheduling",
			slog.String("round_id", fmt.Sprintf("%d", roundID)),
			slog.Any("error", fetchErr),
		)

		errorPayload := roundevents.RoundUpdateErrorPayload{
			RoundUpdateRequest: nil,
			Error:              fmt.Sprintf("failed to fetch round for rescheduling: %v", fetchErr),
		}

		if err := s.publishEvent(msg, roundevents.RoundUpdateError, errorPayload); err != nil {
			return fmt.Errorf("error during error reporting: %w", err)
		}
		return fmt.Errorf("failed to fetch round for rescheduling: %w", fetchErr)
	}

	// Step 4: Use `RoundStoredPayload` instead of creating a custom struct
	storedPayload := roundevents.RoundStoredPayload{
		Round: *round,
	}

	// Step 5: Publish the round update reschedule event
	if err := s.publishEvent(msg, roundevents.RoundUpdateReschedule, storedPayload); err != nil {
		s.logger.Error("Failed to publish event",
			slog.String("event", roundevents.RoundUpdateReschedule),
			slog.String("error", err.Error()),
			slog.String("round_id", fmt.Sprintf("%d", roundID)),
		)

		errorPayload := roundevents.RoundUpdateErrorPayload{
			RoundUpdateRequest: nil,
			Error:              fmt.Sprintf("failed to publish event %s: %v", roundevents.RoundUpdateReschedule, err),
		}

		if err := s.publishEvent(msg, roundevents.RoundUpdateError, errorPayload); err != nil {
			return fmt.Errorf("error during error reporting: %w", err)
		}
		return fmt.Errorf("failed to publish reschedule event: %w", err)
	}

	s.logger.Info("Round reschedule event published successfully",
		slog.String("round_id", fmt.Sprintf("%d", roundID)),
	)

	return nil
}
