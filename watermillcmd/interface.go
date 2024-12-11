package watermillcmd

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
)

// CommandBus defines the interface for a command bus.
type CommandBus interface {
	Send(ctx context.Context, command interface{}) error
	// Add other methods from cqrs.CommandBus as needed
}

// MessageBus defines the interface for publishing and subscribing to messages.
type MessageBus interface {
	PublishEvent(ctx context.Context, topic string, event interface{}) error
	PublishCommand(ctx context.Context, topic string, command interface{}) error
	Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error)
	Close() error
}
