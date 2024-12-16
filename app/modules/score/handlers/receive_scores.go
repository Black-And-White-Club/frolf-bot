package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	scorecommands "github.com/Black-And-White-Club/tcr-bot/app/modules/score/commands"
	scoredb "github.com/Black-And-White-Club/tcr-bot/app/modules/score/db"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/pkg/errors"
)

// ReceiveScoresHandler handles receiving scores from the round module.
type ReceiveScoresHandler struct {
	eventBus watermillutil.PubSuber
}

// NewReceiveScoresHandler creates a new ReceiveScoresHandler.
func NewReceiveScoresHandler(eventBus watermillutil.PubSuber) *ReceiveScoresHandler {
	return &ReceiveScoresHandler{
		eventBus: eventBus,
	}
}

// Handle handles the event or message from the round module.
func (h *ReceiveScoresHandler) Handle(ctx context.Context, msg *message.Message) error {
	var cmd scorecommands.SubmitScoreCommand
	marshaler := watermillutil.Marshaler // Use the marshaler from userhandlers
	if err := marshaler.Unmarshal(msg, &cmd); err != nil {
		return errors.Wrap(err, "failed to unmarshal SubmitScoreCommand")
	}

	// 1. Extract DiscordID, Score, and TagNumber from the received data
	scores := make([]scoredb.Score, len(cmd.Scores))
	for i, s := range cmd.Scores {
		scores[i] = scoredb.Score{
			DiscordID: s.UserID,    // Assuming UserID in cmd.Scores is DiscordID
			RoundID:   cmd.RoundID, // Assuming you have RoundID in the command
			Score:     s.Score,
			TagNumber: s.TagNumber,
		}
	}

	// 2. Publish the scores to be processed by the ScoreProcessor
	payload, err := json.Marshal(scores)
	if err != nil {
		return fmt.Errorf("failed to marshal scores: %w", err)
	}

	if err := h.eventBus.Publish(TopicProcessScores, message.NewMessage(watermill.NewUUID(), payload)); err != nil {
		return fmt.Errorf("failed to publish scores for processing: %w", err)
	}

	return nil
}
