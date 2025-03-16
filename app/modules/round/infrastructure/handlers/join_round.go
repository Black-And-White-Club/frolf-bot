package roundhandlers

import (
	"fmt"
	"log/slog"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
)

func (h *RoundHandlers) HandleRoundParticipantJoinRequest(msg *message.Message) error {
	correlationID, payload, err := eventutil.UnmarshalPayload[roundevents.ParticipantJoinRequestPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal ParticipantJoinRequestPayload: %w", err)
	}

	h.logger.Info("Received ParticipantJoinRequest event",
		slog.String("correlation_id", correlationID),
		slog.String("user_id", string(payload.UserID)),
		slog.String("response", string(payload.Response)),
		slog.Int64("round_id", int64(payload.RoundID)))

	// Log the entire payload for debugging
	h.logger.Debug("Unmarshalled payload",
		slog.String("correlation_id", correlationID),
		slog.Any("payload", payload))

	// Call the CheckParticipantStatus service method
	if err := h.RoundService.CheckParticipantStatus(msg.Context(), msg); err != nil {
		h.logger.Error("Failed to check participant status",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err))
		return err
	}

	return nil
}

func (h *RoundHandlers) HandleRoundParticipantJoinValidationRequest(msg *message.Message) error {
	correlationID, payload, err := eventutil.UnmarshalPayload[roundevents.ParticipantJoinValidationRequestPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal ParticipantJoinValidationRequestPayload: %w", err)
	}

	h.logger.Info("Received ParticipantJoinValidationRequest event",
		slog.String("correlation_id", correlationID),
		slog.Any("payload", payload))

	// Call service method to validate
	if err := h.RoundService.ValidateParticipantJoinRequest(msg.Context(), msg); err != nil {
		h.logger.Error("Failed to validate join request",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err))
		return err
	}

	return nil
}

func (h *RoundHandlers) HandleRoundParticipantRemovalRequest(msg *message.Message) error {
	correlationID, _, err := eventutil.UnmarshalPayload[roundevents.ParticipantRemovalRequestPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal ParticipantRemovalRequestPayload: %w", err)
	}

	h.logger.Info("Received ParticipantRemovalRequest event",
		slog.String("correlation_id", correlationID))

	// Call service method for removal
	if err := h.RoundService.ParticipantRemoval(msg.Context(), msg); err != nil {
		h.logger.Error("Failed to remove participant",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err))
		return err
	}

	return nil
}

// Handle the decline event
func (h *RoundHandlers) HandleRoundParticipantDeclined(msg *message.Message) error {
	correlationID, _, err := eventutil.UnmarshalPayload[roundevents.ParticipantDeclinedPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal ParticipantDeclinedPayload: %w", err)
	}

	h.logger.Info("Received RoundParticipantDeclined event", slog.String("correlation_id", correlationID))

	// Update the participant's status in the database
	if err := h.RoundService.HandleDecline(msg.Context(), msg); err != nil {
		h.logger.Error("Failed to handle decline", slog.String("correlation_id", correlationID), slog.Any("error", err))
		return fmt.Errorf("failed to handle decline: %w", err)
	}

	h.logger.Info("RoundParticipantDeclined event processed", slog.String("correlation_id", correlationID))
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
