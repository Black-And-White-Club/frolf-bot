package scorehandlers

import (
	"context"
	"encoding/json"
	"fmt"

	scoredb "github.com/Black-And-White-Club/tcr-bot/app/modules/score/db"
	scoreservice "github.com/Black-And-White-Club/tcr-bot/app/modules/score/services"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/pkg/errors"
)

// ProcessScoresHandler processes scores and forwards them to the leaderboard module.
type ProcessScoresHandler struct {
	eventBus       watermillutil.PubSuber
	scoreProcessor *scoreservice.ScoresProcessingService
}

// NewProcessScoresHandler creates a new ProcessScoresHandler instance.
func NewProcessScoresHandler(eventBus watermillutil.PubSuber, scoreProcessor *scoreservice.ScoresProcessingService) *ProcessScoresHandler {
	return &ProcessScoresHandler{
		eventBus:       eventBus,
		scoreProcessor: scoreProcessor,
	}
}

// Handle processes the scores received from the ProcessScores topic.
func (h *ProcessScoresHandler) Handle(ctx context.Context, msg *message.Message) error {
	var scores []scoredb.Score
	if err := json.Unmarshal(msg.Payload, &scores); err != nil {
		return errors.Wrap(err, "failed to unmarshal scores for processing")
	}

	// Process the scores using the service
	sortedScores, err := h.scoreProcessor.SortScores(scores)
	if err != nil {
		return errors.Wrap(err, "failed to process scores")
	}

	// Publish sorted scores to the leaderboard topic
	payload, err := json.Marshal(sortedScores)
	if err != nil {
		return fmt.Errorf("failed to marshal sorted scores: %w", err)
	}

	if err := h.eventBus.Publish(TopicSendToLeaderboard, message.NewMessage(watermill.NewUUID(), payload)); err != nil {
		return fmt.Errorf("failed to publish sorted scores to leaderboard: %w", err)
	}

	return nil
}
