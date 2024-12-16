package scorehandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	scoredb "github.com/Black-And-White-Club/tcr-bot/app/modules/score/db"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/pkg/errors"
)

type ReceivedScore struct {
	UserID    string  `json:"user_id"`
	Score     float64 `json:"score"`
	TagNumber string  `json:"tag_number"`
}

type ReceivedScoresMessage struct {
	RoundID string          `json:"round_id"`
	Scores  []ReceivedScore `json:"scores"`
}

// ReceiveScoresHandler handles receiving scores from the round module.
type ReceiveScoresHandler struct {
	eventBus watermillutil.PubSuber
}

// NewReceiveScoresHandler creates a new ReceiveScoresHandler.
func NewReceiveScoresHandler(eventBus watermillutil.PubSuber) *ReceiveScoresHandler {
	return &ReceiveScoresHandler{eventBus: eventBus}
}

// Handle handles the event or message from the round module.
func (h *ReceiveScoresHandler) Handle(ctx context.Context, msg *message.Message) error {
	var receivedScores ReceivedScoresMessage
	marshaler := watermillutil.Marshaler
	if err := marshaler.Unmarshal(msg, &receivedScores); err != nil {
		return errors.Wrap(err, "failed to unmarshal incoming scores")
	}

	scores := make([]scoredb.Score, len(receivedScores.Scores))
	for i, s := range receivedScores.Scores {
		// Convert score while preserving negatives
		scoreValue := int(s.Score)

		// Convert tag number with error handling
		tagNumber, err := strconv.Atoi(s.TagNumber)
		if err != nil {
			return fmt.Errorf("failed to convert tag number %q for user %q at index %d: %w", s.TagNumber, s.UserID, i, err)
		}

		// Create the Score struct
		scores[i] = scoredb.Score{
			DiscordID: s.UserID,
			RoundID:   receivedScores.RoundID,
			Score:     scoreValue,
			TagNumber: tagNumber,
		}
	}

	// 1. Publish ScoresReceivedEvent
	event := ScoresReceivedEvent{
		RoundID: receivedScores.RoundID,
		Scores:  scores, // This is now correct
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal ScoresReceivedEvent: %w", err)
	}

	if err := h.eventBus.Publish(TopicReceiveScores, message.NewMessage(watermill.NewUUID(), payload)); err != nil {
		return fmt.Errorf("failed to publish ScoresReceivedEvent: %w", err)
	}

	return nil
}
