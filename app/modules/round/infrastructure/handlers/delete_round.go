package roundhandlers

import (
	"fmt"
	"log/slog"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
)

func (h *RoundHandlers) HandleRoundDeleteRequest(msg *message.Message) error {
	correlationID, _, err := eventutil.UnmarshalPayload[roundevents.RoundDeleteRequestPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal RoundDeleteRequestPayload: %w", err)
	}

	h.logger.Info("Received RoundDeleteRequest event",
		slog.String("correlation_id", correlationID),
	)

	if err := h.RoundService.ValidateRoundDeleteRequest(msg.Context(), msg); err != nil {
		h.logger.Error("Failed to handle RoundDeleteRequest event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to handle RoundDeleteRequest event: %w", err)
	}

	h.logger.Info("RoundDeleteRequest event processed", slog.String("correlation_id", correlationID))
	return nil
}

func (h *RoundHandlers) HandleRoundDeleteValidated(msg *message.Message) error {
	correlationID, _, err := eventutil.UnmarshalPayload[roundevents.RoundDeleteValidatedPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal RoundDeleteValidatedPayload: %w", err)
	}

	h.logger.Info("Received RoundDeleteValidated event",
		slog.String("correlation_id", correlationID),
	)

	if err := h.RoundService.CheckRoundExists(msg.Context(), msg); err != nil {
		h.logger.Error("Failed to handle RoundDeleteValidated event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to handle RoundDeleteValidated event: %w", err)
	}

	h.logger.Info("RoundDeleteValidated event processed", slog.String("correlation_id", correlationID))
	return nil
}

func (h *RoundHandlers) HandleRoundToDeleteFetched(msg *message.Message) error {
	correlationID, _, err := eventutil.UnmarshalPayload[roundevents.RoundToDeleteFetchedPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal RoundToDeleteFetchedPayload: %w", err)
	}

	h.logger.Info("Received RoundToDeleteFetched event",
		slog.String("correlation_id", correlationID),
	)

	if err := h.RoundService.CheckUserAuthorization(msg.Context(), msg); err != nil {
		h.logger.Error("Failed to handle RoundToDeleteFetched event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to handle RoundToDeleteFetched event: %w", err)
	}

	h.logger.Info("RoundToDeleteFetched event processed", slog.String("correlation_id", correlationID))
	return nil
}

func (h *RoundHandlers) HandleRoundDeleteAuthorized(msg *message.Message) error {
	correlationID, _, err := eventutil.UnmarshalPayload[roundevents.RoundDeleteAuthorizedPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal RoundDeleteAuthorizedPayload: %w", err)
	}

	h.logger.Info("Received RoundDeleteAuthorized event",
		slog.String("correlation_id", correlationID),
	)

	if err := h.RoundService.DeleteRound(msg.Context(), msg); err != nil {
		h.logger.Error("Failed to handle RoundDeleteAuthorized event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to handle RoundDeleteAuthorized event: %w", err)
	}

	h.logger.Info("RoundDeleteAuthorized event processed", slog.String("correlation_id", correlationID))
	return nil
}

func (h *RoundHandlers) HandleRoundUserRoleCheckResult(msg *message.Message) error {
	correlationID, _, err := eventutil.UnmarshalPayload[roundevents.UserRoleCheckResultPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal UserRoleCheckResultPayload: %w", err)
	}

	h.logger.Info("Received RoundUserRoleCheckResult event",
		slog.String("correlation_id", correlationID),
	)

	if err := h.RoundService.UserRoleCheckResult(msg.Context(), msg); err != nil {
		h.logger.Error("Failed to handle RoundUserRoleCheckResult event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to handle RoundUserRoleCheckResult event: %w", err)
	}

	h.logger.Info("RoundUserRoleCheckResult event processed", slog.String("correlation_id", correlationID))
	return nil
}
