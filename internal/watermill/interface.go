package watermillutil

import (
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
)

// CommandBus defines the interface for a command bus.
type CommandBus interface {
	cqrs.CommandBus
}

// MessageBus defines the interface for publishing and subscribing to messages.
type MessageBus interface {
	Publish(topic string, msg *message.Message) error
	Subscribe(topic string) (<-chan *message.Message, error)
	Close() error
}
