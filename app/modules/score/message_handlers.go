package score

import (
	"context"
	"encoding/json"
	"fmt"

	scoreevents "github.com/Black-And-White-Club/tcr-bot/app/modules/score/events"
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
	case scoreevents.ScoreCorrectedEventSubject:
		return h.handleScoreCorrection(ctx, msg)
	// ... handle other message types based on subject ...
	default:
		h.logger.Error("Unknown message type", fmt.Errorf("unknown message type: %s", subject), watermill.LogFields{
			"subject": subject,
		})
		return fmt.Errorf("unknown message type: %s", subject)
	}
}

func (h *MessageHandlers) handleScoreCorrection(_ context.Context, msg *message.Message) error {
	var correctionEvent scoreevents.ScoreCorrectedEvent
	if err := json.Unmarshal(msg.Payload, &correctionEvent); err != nil {
		h.logger.Error("Failed to unmarshal ScoreCorrectionEvent", err, watermill.LogFields{
			"message_id": msg.UUID,
		})
		return fmt.Errorf("failed to unmarshal ScoreCorrectionEvent: %w", err)
	}

	// 2. Publish ScoreCorrectionEvent
	eventData, err := json.Marshal(correctionEvent)
	if err != nil {
		h.logger.Error("Failed to marshal ScoreCorrectionEvent", err, watermill.LogFields{
			"message_id": msg.UUID,
		})
		return fmt.Errorf("failed to marshal ScoreCorrectionEvent: %w", err)
	}

	if err := h.Publisher.Publish(scoreevents.ScoreCorrectedEventSubject, message.NewMessage(watermill.NewUUID(), eventData)); err != nil {
		h.logger.Error("Failed to publish ScoreCorrectionEvent", err, watermill.LogFields{
			"message_id": msg.UUID,
		})
		return fmt.Errorf("failed to publish ScoreCorrectionEvent: %w", err)
	}
	msg.Ack()
	return nil
}
