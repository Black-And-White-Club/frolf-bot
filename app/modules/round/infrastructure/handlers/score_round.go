package roundhandlers

import (
	"fmt"
	"log/slog"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
)

func (h *RoundHandlers) HandleRoundScoreUpdateRequest(msg *message.Message) error {
	correlationID, _, err := eventutil.UnmarshalPayload[roundevents.ScoreUpdateRequestPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal RoundScoreUpdateRequestPayload: %w", err)
	}

	h.logger.Info("Received RoundScoreUpdateRequest event",
		slog.String("correlation_id", correlationID),
	)

	if err := h.RoundService.ValidateScoreUpdateRequest(msg.Context(), msg); err != nil {
		h.logger.Error("Failed to handle RoundScoreUpdateRequest event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to handle RoundScoreUpdateRequest event: %w", err)
	}

	h.logger.Info("RoundScoreUpdateRequest event processed", slog.String("correlation_id", correlationID))
	return nil
}

func (h *RoundHandlers) HandleRoundScoreUpdateValidated(msg *message.Message) error {
	correlationID, _, err := eventutil.UnmarshalPayload[roundevents.ScoreUpdateValidatedPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal RoundScoreUpdateValidatedPayload: %w", err)
	}

	h.logger.Info("Received RoundScoreUpdateValidated event",
		slog.String("correlation_id", correlationID),
	)

	if err := h.RoundService.UpdateParticipantScore(msg.Context(), msg); err != nil {
		h.logger.Error("Failed to handle RoundScoreUpdateValidated event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to handle RoundScoreUpdateValidated event: %w", err)
	}

	h.logger.Info("RoundScoreUpdateValidated event processed", slog.String("correlation_id", correlationID))
	return nil
}

func (h *RoundHandlers) HandleRoundParticipantScoreUpdated(msg *message.Message) error {
	correlationID, _, err := eventutil.UnmarshalPayload[roundevents.ParticipantScoreUpdatedPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal RoundParticipantScoreUpdatedPayload: %w", err)
	}

	h.logger.Info("Received RoundParticipantScoreUpdated event",
		slog.String("correlation_id", correlationID),
	)

	if err := h.RoundService.CheckAllScoresSubmitted(msg.Context(), msg); err != nil {
		h.logger.Error("Failed to handle RoundParticipantScoreUpdated event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to handle RoundParticipantScoreUpdated event: %w", err)
	}

	h.logger.Info("RoundParticipantScoreUpdated event processed", slog.String("correlation_id", correlationID))
	return nil
}

func (h *RoundHandlers) HandleRoundAllScoresSubmitted(msg *message.Message) error {
	correlationID, _, err := eventutil.UnmarshalPayload[roundevents.AllScoresSubmittedPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal RoundAllScoresSubmittedPayload: %w", err)
	}

	h.logger.Info("Received RoundAllScoresSubmitted event",
		slog.String("correlation_id", correlationID),
	)

	if err := h.RoundService.FinalizeRound(msg.Context(), msg); err != nil {
		h.logger.Error("Failed to handle RoundAllScoresSubmitted event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to handle RoundAllScoresSubmitted event: %w", err)
	}

	h.logger.Info("RoundAllScoresSubmitted event processed", slog.String("correlation_id", correlationID))
	return nil
}

func (h *RoundHandlers) HandleRoundFinalized(msg *message.Message) error {
	correlationID, _, err := eventutil.UnmarshalPayload[roundevents.RoundFinalizedPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal RoundFinalizedPayload: %w", err)
	}

	h.logger.Info("Received RoundFinalized event",
		slog.String("correlation_id", correlationID),
	)

	if err := h.RoundService.NotifyScoreModule(msg.Context(), msg); err != nil {
		h.logger.Error("Failed to handle RoundFinalized event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to handle RoundFinalized event: %w", err)
	}

	h.logger.Info("RoundFinalized event processed", slog.String("correlation_id", correlationID))
	return nil
}
