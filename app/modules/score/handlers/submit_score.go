package scorehandlers

import (
	"context"
	"encoding/json"
	"fmt"

	scoredb "github.com/Black-And-White-Club/tcr-bot/app/modules/score/db"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/pkg/errors"
)

// SubmitScoresHandler handles submitting scores to the database and forwarding them to the leaderboard module.
type SubmitScoresHandler struct {
	eventBus watermillutil.PubSuber
	scoredb  scoredb.ScoreDB
}

// NewSubmitScoresHandler creates a new SubmitScoresHandler instance.
func NewSubmitScoresHandler(eventBus watermillutil.PubSuber, scoredb scoredb.ScoreDB) *SubmitScoresHandler {
	return &SubmitScoresHandler{
		eventBus: eventBus,
		scoredb:  scoredb,
	}
}

// Handle submits the scores to the database and forwards them to the leaderboard module.
func (h *SubmitScoresHandler) Handle(ctx context.Context, msg *message.Message) error {
	var scores []scoredb.Score // Corrected the type here
	if err := json.Unmarshal(msg.Payload, &scores); err != nil {
		return errors.Wrap(err, "failed to unmarshal sorted scores")
	}

	// Insert scores into the database
	if err := h.scoredb.InsertScores(ctx, scores); err != nil {
		return fmt.Errorf("failed to insert scores into the database: %w", err)
	}

	// Publish scores to the leaderboard module
	payload, err := json.Marshal(scores)
	if err != nil {
		return fmt.Errorf("failed to marshal scores for leaderboard: %w", err)
	}

	if err := h.eventBus.Publish(TopicSendToLeaderboard, message.NewMessage(watermill.NewUUID(), payload)); err != nil {
		return fmt.Errorf("failed to publish scores to leaderboard: %w", err)
	}

	return nil
}
