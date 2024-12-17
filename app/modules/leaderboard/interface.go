package leaderboard

import (
	"github.com/ThreeDotsLabs/watermill/message"
)

// MessageHandler defines the interface for handling messages.
type MessageHandler interface {
	Handle(msg *message.Message) ([]*message.Message, error)
}
