package roundhandlers

import (
	"fmt"
	"log/slog"

	roundevents "github.com/Black-And-White-Club/tcr-bot/app/modules/round/domain/events"
	"github.com/Black-And-White-Club/tcr-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
)

func (h *RoundHandlers) HandleRoundCreateRequest(msg *message.Message) error {
	correlationID, _, err := eventutil.UnmarshalPayload[roundevents.RoundCreateRequestPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal RoundCreateRequestPayload: %w", err)
	}

	h.logger.Info("Received RoundCreateRequest event",
		slog.String("correlation_id", correlationID),
	)

	if err := h.RoundService.ValidateRoundRequest(msg.Context(), msg); err != nil {
		h.logger.Error("Failed to handle RoundCreateRequest event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to handle RoundCreateRequest event: %w", err)
	}

	h.logger.Info("RoundCreateRequest event processed", slog.String("correlation_id", correlationID))
	return nil
}

func (h *RoundHandlers) HandleRoundValidated(msg *message.Message) error {
	correlationID, _, err := eventutil.UnmarshalPayload[roundevents.RoundValidatedPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal RoundValidatedPayload: %w", err)
	}

	h.logger.Info("Received RoundValidated event",
		slog.String("correlation_id", correlationID),
	)

	if err := h.RoundService.ParseDateTime(msg.Context(), msg); err != nil {
		h.logger.Error("Failed to handle RoundValidated event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to handle RoundValidated event: %w", err)
	}

	h.logger.Info("RoundValidated event processed", slog.String("correlation_id", correlationID))
	return nil
}

func (h *RoundHandlers) HandleRoundDateTimeParsed(msg *message.Message) error {
	correlationID, _, err := eventutil.UnmarshalPayload[roundevents.RoundDateTimeParsedPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal RoundDateTimeParsedPayload: %w", err)
	}

	h.logger.Info("Received RoundDateTimeParsed event",
		slog.String("correlation_id", correlationID),
	)

	if err := h.RoundService.CreateRoundEntity(msg.Context(), msg); err != nil {
		h.logger.Error("Failed to handle RoundDateTimeParsed event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to handle RoundDateTimeParsed event: %w", err)
	}

	h.logger.Info("RoundDateTimeParsed event processed", slog.String("correlation_id", correlationID))
	return nil
}

func (h *RoundHandlers) HandleRoundEntityCreated(msg *message.Message) error {
	correlationID, _, err := eventutil.UnmarshalPayload[roundevents.RoundEntityCreatedPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal RoundEntityCreatedPayload: %w", err)
	}

	h.logger.Info("Received RoundEntityCreated event",
		slog.String("correlation_id", correlationID),
	)

	if err := h.RoundService.StoreRound(msg.Context(), msg); err != nil {
		h.logger.Error("Failed to handle RoundEntityCreated event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to handle RoundEntityCreated event: %w", err)
	}

	h.logger.Info("RoundEntityCreated event processed", slog.String("correlation_id", correlationID))
	return nil
}

func (h *RoundHandlers) HandleRoundStored(msg *message.Message) error {
	correlationID, _, err := eventutil.UnmarshalPayload[roundevents.RoundStoredPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal RoundStoredPayload: %w", err)
	}

	h.logger.Info("Received RoundStored event",
		slog.String("correlation_id", correlationID),
	)

	if err := h.RoundService.ScheduleRoundEvents(msg.Context(), msg); err != nil {
		h.logger.Error("Failed to handle RoundStored event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to handle RoundStored event: %w", err)
	}

	h.logger.Info("RoundStored event processed", slog.String("correlation_id", correlationID))
	return nil
}

func (h *RoundHandlers) HandleRoundScheduled(msg *message.Message) error {
	correlationID, _, err := eventutil.UnmarshalPayload[roundevents.RoundScheduledPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal RoundScheduledPayload: %w", err)
	}

	h.logger.Info("Received RoundScheduled event",
		slog.String("correlation_id", correlationID),
	)

	if err := h.RoundService.PublishRoundCreated(msg.Context(), msg); err != nil {
		h.logger.Error("Failed to handle RoundScheduled event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to handle RoundScheduled event: %w", err)
	}

	h.logger.Info("RoundScheduled event processed", slog.String("correlation_id", correlationID))
	return nil
}
