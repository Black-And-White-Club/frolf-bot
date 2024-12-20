package scorehandlers

import (
	"context"
	"encoding/json"
	"fmt"

	scoreevents "github.com/Black-And-White-Club/tcr-bot/app/modules/score/events"
	scoreservice "github.com/Black-And-White-Club/tcr-bot/app/modules/score/services"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// ScoreHandlers struct to hold dependencies
type ScoreHandlers struct {
	ScoreService scoreservice.Service
	Publisher    message.Publisher
	logger       watermill.LoggerAdapter
}

// NewScoreHandlers creates a new ScoreHandlers instance.
func NewScoreHandlers(service scoreservice.Service, publisher message.Publisher, logger watermill.LoggerAdapter) *ScoreHandlers {
	return &ScoreHandlers{
		ScoreService: service,
		Publisher:    publisher,
		logger:       logger,
	}
}

// HandleScoreCorrected handles the ScoreCorrectedEvent.
func (h *ScoreHandlers) HandleScoreCorrected(ctx context.Context, msg *message.Message) error {
	defer msg.Ack()

	var event scoreevents.ScoreCorrectedEvent
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		return fmt.Errorf("failed to unmarshal ScoreCorrectedEvent: %w", err)
	}

	if err := h.ScoreService.CorrectScore(context.Background(), event); err != nil {
		return fmt.Errorf("failed to correct score: %w", err)
	}

	return nil
}
