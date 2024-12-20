package leaderboard

import (
	"context"
	"encoding/json"
	"fmt"

	leaderboardevents "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/events"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// MessageHandlers handles incoming messages and publishes corresponding events.
type MessageHandlers struct {
	Publisher message.Publisher
	logger    watermill.LoggerAdapter
}

// NewMessageHandlers creates a new MessageHandlers.
func NewMessageHandlers(publisher message.Publisher, logger watermill.LoggerAdapter) *MessageHandlers {
	return &MessageHandlers{
		Publisher: publisher,
		logger:    logger,
	}
}

// HandleMessage processes incoming messages and publishes corresponding events.
func (h *MessageHandlers) HandleMessage(msg *message.Message) error {
	// 1. Determine message type based on subject.
	subject := msg.Metadata.Get("subject")
	ctx := context.Background()

	switch subject {
	case leaderboardevents.TagSwapRequestSubject:
		return h.handleTagSwapRequest(ctx, msg)
	// ... handle other message types based on subject ...
	default:
		h.logger.Error("Unknown message type", fmt.Errorf("unknown message type: %s", subject), watermill.LogFields{
			"subject": subject,
		})
		return fmt.Errorf("unknown message type: %s", subject)
	}
}

func (h *MessageHandlers) handleTagSwapRequest(_ context.Context, msg *message.Message) error {
	var swapRequest leaderboardevents.TagSwapRequest // Use the TagSwapRequest from events
	if err := json.Unmarshal(msg.Payload, &swapRequest); err != nil {
		h.logger.Error("Failed to unmarshal TagSwapRequest", err, watermill.LogFields{
			"message_id": msg.UUID,
		})
		return fmt.Errorf("failed to unmarshal TagSwapRequest: %w", err)
	}

	// 2. Publish TagSwapRequestEvent
	eventData, err := json.Marshal(&swapRequest)
	if err != nil {
		h.logger.Error("Failed to marshal TagSwapRequest", err, watermill.LogFields{
			"message_id": msg.UUID,
		})
		return fmt.Errorf("failed to marshal TagSwapRequest: %w", err)
	}

	if err := h.Publisher.Publish(leaderboardevents.TagSwapRequestSubject, message.NewMessage(watermill.NewUUID(), eventData)); err != nil {
		h.logger.Error("Failed to publish TagSwapRequest", err, watermill.LogFields{
			"message_id": msg.UUID,
		})
		return fmt.Errorf("failed to publish TagSwapRequest: %w", err)
	}
	msg.Ack()
	return nil
}
