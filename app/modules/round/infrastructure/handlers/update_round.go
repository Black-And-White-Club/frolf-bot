package roundhandlers

import (
	"fmt"
	"log/slog"
	"strconv"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
)

func (h *RoundHandlers) HandleRoundUpdateRequest(msg *message.Message) error {
	correlationID, _, err := eventutil.UnmarshalPayload[roundevents.RoundUpdateRequestPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal RoundUpdateRequestPayload: %w", err)
	}

	h.logger.Info("Received RoundUpdateRequest event",
		slog.String("correlation_id", correlationID),
	)
	if err := h.RoundService.ValidateRoundUpdateRequest(msg.Context(), msg); err != nil {
		h.logger.Error("Failed to handle RoundUpdateRequest event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to handle RoundUpdateRequest event: %w", err)
	}

	h.logger.Info("RoundUpdateRequest event processed", slog.String("correlation_id", correlationID))
	return nil
}

func (h *RoundHandlers) HandleRoundUpdateValidated(msg *message.Message) error {
	correlationID, _, err := eventutil.UnmarshalPayload[roundevents.RoundUpdateValidatedPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal RoundUpdateValidatedPayload: %w", err)
	}

	h.logger.Info("Received RoundUpdateValidated event",
		slog.String("correlation_id", correlationID),
	)
	if err := h.RoundService.GetRound(msg.Context(), msg); err != nil {
		h.logger.Error("Failed to handle RoundUpdateValidated event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to handle RoundUpdateValidated event: %w", err)
	}

	h.logger.Info("RoundUpdateValidated event processed", slog.String("correlation_id", correlationID))
	return nil
}

func (h *RoundHandlers) HandleRoundFetched(msg *message.Message) error {
	correlationID, _, err := eventutil.UnmarshalPayload[roundevents.RoundFetchedPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal RoundFetchedPayload: %w", err)
	}

	h.logger.Info("Received RoundFetched event",
		slog.String("correlation_id", correlationID),
	)
	if err := h.RoundService.UpdateRoundEntity(msg.Context(), msg); err != nil {
		h.logger.Error("Failed to handle RoundFetched event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to handle RoundFetched event: %w", err)
	}

	h.logger.Info("RoundFetched event processed", slog.String("correlation_id", correlationID))
	return nil
}

func (h *RoundHandlers) HandleRoundEntityUpdated(msg *message.Message) error {
	correlationID, _, err := eventutil.UnmarshalPayload[roundevents.RoundEntityUpdatedPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal RoundEntityUpdatedPayload: %w", err)
	}

	h.logger.Info("Received RoundEntityUpdated event",
		slog.String("correlation_id", correlationID),
	)
	if err := h.RoundService.StoreRoundUpdate(msg.Context(), msg); err != nil {
		h.logger.Error("Failed to handle RoundEntityUpdated event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to handle RoundEntityUpdated event: %w", err)
	}

	h.logger.Info("RoundEntityUpdated event processed", slog.String("correlation_id", correlationID))
	return nil
}

// HandleRoundScheduleUpdate handles the round.schedule.update event.
func (h *RoundHandlers) HandleRoundScheduleUpdate(msg *message.Message) error {
	correlationID, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundScheduleUpdatePayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal RoundScheduleUpdatePayload: %w", err)
	}

	// Convert int64 RoundID to string
	roundIDStr := strconv.FormatInt(eventPayload.RoundID, 10)

	h.logger.Info("Received RoundScheduleUpdate event",
		slog.String("correlation_id", correlationID),
		slog.String("round_id", roundIDStr), // Use the converted string
	)

	// Update the scheduled events for the round
	if err := h.RoundService.UpdateScheduledRoundEvents(msg.Context(), msg); err != nil {
		h.logger.Error("Failed to handle RoundScheduleUpdate event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to handle RoundScheduleUpdate event: %w", err)
	}

	h.logger.Info("RoundScheduleUpdate event processed", slog.String("correlation_id", correlationID))
	return nil
}
