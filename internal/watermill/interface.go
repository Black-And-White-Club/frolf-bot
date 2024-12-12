package watermillutil

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
)

// CommandBus defines the interface for a command bus.
type CommandBus interface {
	cqrs.CommandBus
}

// Publisher defines the interface for publishing messages.
type Publisher interface {
	Publish(topic string, messages ...*message.Message) error
	Close() error
}

// Subscriber defines the interface for subscribing to messages.
type Subscriber interface {
	Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error)
	Close() error
}

// PubSub combines the Publisher and Subscriber interfaces.
type PubSuber interface {
	Publisher
	Subscriber
}
