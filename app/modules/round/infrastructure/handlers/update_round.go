package roundhandlers

import (
	"fmt"
	"log/slog"

	roundevents "github.com/Black-And-White-Club/tcr-bot/app/modules/round/domain/events"
	"github.com/Black-And-White-Club/tcr-bot/internal/eventutil"
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

func (h *RoundHandlers) HandleRoundUpdated(msg *message.Message) error {
	correlationID, _, err := eventutil.UnmarshalPayload[roundevents.RoundUpdatedPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal RoundUpdatedPayload: %w", err)
	}

	h.logger.Info("Received RoundUpdated event",
		slog.String("correlation_id", correlationID),
	)
	if err := h.RoundService.PublishRoundUpdated(msg.Context(), msg); err != nil {
		h.logger.Error("Failed to handle RoundUpdated event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to handle RoundUpdated event: %w", err)
	}

	h.logger.Info("RoundUpdated event processed", slog.String("correlation_id", correlationID))
	return nil
}
