package roundhandlers

import (
	"context"
	"encoding/json"
	"fmt"

	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// SendScoresHandler handles sending scores to the score module.
type SendScoresHandler struct {
	PubSuber watermillutil.PubSuber
}

// NewSendScoresHandler creates a new SendScoresHandler.
func NewSendScoresHandler(pubsuber watermillutil.PubSuber) *SendScoresHandler {
	return &SendScoresHandler{
		PubSuber: pubsuber,
	}
}

// Handle processes the SendScoresEvent.
func (h *SendScoresHandler) Handle(ctx context.Context, msg *message.Message) error {
	var event SendScoresEvent
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		return fmt.Errorf("failed to unmarshal SendScoresEvent: %w", err)
	}

	// Construct the payload for the score module
	scoreModulePayload := map[string]interface{}{
		"round_id": event.RoundID,
		"scores":   event.Scores,
	}

	// Marshal the payload into JSON
	jsonPayload, err := json.Marshal(scoreModulePayload)
	if err != nil {
		return fmt.Errorf("failed to marshal score module payload: %w", err)
	}

	// Publish the payload to the score module (using the PubSuber)
	if err := h.PubSuber.Publish("score.module.submit", message.NewMessage(watermill.NewUUID(), jsonPayload)); err != nil { // Replace "score.module.submit" with your actual topic
		return fmt.Errorf("failed to publish to score module: %w", err)
	}

	return nil
}
