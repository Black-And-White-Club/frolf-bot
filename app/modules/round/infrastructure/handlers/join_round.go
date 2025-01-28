package roundhandlers

import (
	"fmt"
	"log/slog"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
)

func (h *RoundHandlers) HandleRoundParticipantJoinRequest(msg *message.Message) error {
	correlationID, _, err := eventutil.UnmarshalPayload[roundevents.ParticipantJoinRequestPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal ParticipantJoinRequestPayload: %w", err)
	}

	h.logger.Info("Received ParticipantJoinRequest event",
		slog.String("correlation_id", correlationID),
	)

	if err := h.RoundService.ValidateParticipantJoinRequest(msg.Context(), msg); err != nil {
		h.logger.Error("Failed to handle ParticipantJoinRequest event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to handle ParticipantJoinRequest event: %w", err)
	}

	h.logger.Info("ParticipantJoinRequest event processed", slog.String("correlation_id", correlationID))
	return nil
}

func (h *RoundHandlers) HandleRoundParticipantJoinValidated(msg *message.Message) error {
	correlationID, _, err := eventutil.UnmarshalPayload[roundevents.ParticipantJoinValidatedPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal ParticipantJoinValidatedPayload: %w", err)
	}

	h.logger.Info("Received ParticipantJoinValidated event",
		slog.String("correlation_id", correlationID),
	)

	if err := h.RoundService.CheckParticipantTag(msg.Context(), msg); err != nil {
		h.logger.Error("Failed to handle ParticipantJoinValidated event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to handle ParticipantJoinValidated event: %w", err)
	}

	h.logger.Info("ParticipantJoinValidated event processed", slog.String("correlation_id", correlationID))
	return nil
}

func (h *RoundHandlers) HandleRoundTagNumberFound(msg *message.Message) error {
	correlationID, _, err := eventutil.UnmarshalPayload[roundevents.RoundTagNumberFoundPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal RoundTagNumberFoundPayload: %w", err)
	}

	h.logger.Info("Received RoundTagNumberFound event",
		slog.String("correlation_id", correlationID),
	)

	if err := h.RoundService.ParticipantTagFound(msg.Context(), msg); err != nil {
		h.logger.Error("Failed to handle RoundTagNumberFound event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to handle RoundTagNumberFound event: %w", err)
	}

	h.logger.Info("RoundTagNumberFound event processed", slog.String("correlation_id", correlationID))
	return nil
}

func (h *RoundHandlers) HandleRoundTagNumberNotFound(msg *message.Message) error {
	correlationID, _, err := eventutil.UnmarshalPayload[roundevents.RoundTagNumberNotFoundPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal RoundTagNumberNotFoundPayload: %w", err)
	}

	h.logger.Info("Received RoundTagNumberNotFound event",
		slog.String("correlation_id", correlationID),
	)

	if err := h.RoundService.ParticipantTagNotFound(msg.Context(), msg); err != nil {
		h.logger.Error("Failed to handle RoundTagNumberNotFound event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to handle RoundTagNumberNotFound event: %w", err)
	}

	h.logger.Info("RoundTagNumberNotFound event processed", slog.String("correlation_id", correlationID))
	return nil
}
