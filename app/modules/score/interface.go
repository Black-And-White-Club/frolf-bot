package score

import (
	"github.com/ThreeDotsLabs/watermill/message"
)

// MessageHandler handles incoming messages.
type MessageHandler interface {
	HandleMessage(msg *message.Message) error
}
