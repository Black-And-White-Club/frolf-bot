package scorehandlers

import (
	"fmt"
	"log/slog"

	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	scoreservice "github.com/Black-And-White-Club/frolf-bot/app/modules/score/application"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
)

type ScoreHandlers struct {
	scoreService scoreservice.Service
	logger       *slog.Logger
}

func NewScoreHandlers(scoreService scoreservice.Service, logger *slog.Logger) Handlers {
	return &ScoreHandlers{
		scoreService: scoreService,
		logger:       logger,
	}
}

func (h *ScoreHandlers) HandleProcessRoundScoresRequest(msg *message.Message) error {
	correlationID, payload, err := eventutil.UnmarshalPayload[scoreevents.ProcessRoundScoresRequestPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal ProcessRoundScoresRequestPayload: %w", err)
	}

	h.logger.Info("Received ProcessRoundScoresRequest event",
		slog.String("correlation_id", correlationID),
		slog.String("round_id", payload.RoundID),
	)

	if err := h.scoreService.ProcessRoundScores(msg.Context(), payload); err != nil {
		h.logger.Error("Failed to handle ProcessRoundScoresRequest event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to handle ProcessRoundScoresRequest event: %w", err)
	}

	h.logger.Info("ProcessRoundScoresRequest event processed", slog.String("correlation_id", correlationID))
	return nil
}

func (h *ScoreHandlers) HandleScoreUpdateRequest(msg *message.Message) error {
	correlationID, payload, err := eventutil.UnmarshalPayload[scoreevents.ScoreUpdateRequestPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal ScoreUpdateRequestPayload: %w", err)
	}

	h.logger.Info("Received ScoreUpdateRequest event",
		slog.String("correlation_id", correlationID),
		slog.String("round_id", payload.RoundID),
	)

	if err := h.scoreService.CorrectScore(msg.Context(), payload); err != nil {
		h.logger.Error("Failed to handle ScoreUpdateRequest event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to handle ScoreUpdateRequest event: %w", err)
	}

	h.logger.Info("ScoreUpdateRequest event processed", slog.String("correlation_id", correlationID))
	return nil
}
