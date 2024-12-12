package user

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
)

// MessageHandler is an interface for handlers that process messages.
type MessageHandler interface {
	Handle(ctx context.Context, msg *message.Message) ([]*message.Message, error)
}
