package score

import (
	"github.com/ThreeDotsLabs/watermill/message"
)

// MessageHandler is an interface for handlers that process messages.
type MessageHandler interface {
	Handle(msg *message.Message) ([]*message.Message, error)
}
