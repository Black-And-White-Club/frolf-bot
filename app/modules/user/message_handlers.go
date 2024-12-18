package user

import (
	"encoding/json"
	"fmt"

	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/events"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// MessageHandlers handles incoming messages and publishes corresponding events.
type MessageHandlers struct {
	Publisher message.Publisher
}

// HandleMessage processes incoming messages and publishes corresponding events.
func (h *MessageHandlers) HandleMessage(msg *message.Message) error {
	// 1. Determine message type (e.g., from metadata or payload)
	messageType := msg.Metadata.Get("type")

	switch messageType {
	case "user_signup":
		var signupReq struct {
			DiscordID string `json:"discord_id"`
			TagNumber int    `json:"tag_number"`
		}
		if err := json.Unmarshal(msg.Payload, &signupReq); err != nil {
			return fmt.Errorf("failed to unmarshal signup request: %w", err)
		}

		// 2. Publish UserSignupRequest event
		event := userevents.UserSignupRequest{
			DiscordID: signupReq.DiscordID,
			TagNumber: signupReq.TagNumber,
		}
		eventData, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("failed to marshal UserSignupRequest: %w", err)
		}
		if err := h.Publisher.Publish(userevents.UserSignupRequestSubject, message.NewMessage(watermill.NewUUID(), eventData)); err != nil {
			return fmt.Errorf("failed to publish UserSignupRequest: %w", err)
		}

	// ... handle other message types ...

	default:
		return fmt.Errorf("unknown message type: %s", messageType)
	}

	return nil
}
