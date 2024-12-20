package user

import (
	"context"
	"encoding/json"
	"fmt"

	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/events"
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
	subject := msg.Metadata.Get("subject") // Assuming you set the subject in the metadata
	ctx := context.Background()

	switch subject {
	case userevents.UserSignupRequestSubject:
		return h.handleUserSignupRequest(ctx, msg)

		// ... handle other message types based on subject ...

	default:
		h.logger.Error("Unknown message type", fmt.Errorf("unknown message type: %s", subject), watermill.LogFields{
			"subject": subject,
		})
		return fmt.Errorf("unknown message type: %s", subject)
	}
}

func (h *MessageHandlers) handleUserSignupRequest(_ context.Context, msg *message.Message) error {
	var signupReq userevents.UserSignupRequest
	if err := json.Unmarshal(msg.Payload, &signupReq); err != nil {
		h.logger.Error("Failed to unmarshal UserSignupRequest", err, watermill.LogFields{
			"message_id": msg.UUID,
		})
		return fmt.Errorf("failed to unmarshal signup request: %w", err)
	}

	// 2. Publish UserSignupRequest event
	event := userevents.UserSignupRequest{
		DiscordID: signupReq.DiscordID,
		TagNumber: signupReq.TagNumber,
	}
	eventData, err := json.Marshal(event)
	if err != nil {
		h.logger.Error("Failed to marshal UserSignupRequest", err, watermill.LogFields{
			"message_id": msg.UUID,
		})
		return fmt.Errorf("failed to marshal UserSignupRequest: %w", err)
	}

	if err := h.Publisher.Publish(userevents.UserSignupRequestSubject, message.NewMessage(watermill.NewUUID(), eventData)); err != nil {
		h.logger.Error("Failed to publish UserSignupRequest", err, watermill.LogFields{
			"message_id": msg.UUID,
		})
		return fmt.Errorf("failed to publish UserSignupRequest: %w", err)
	}
	msg.Ack()
	return nil
}
