package shared

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
)

// EventBus defines the interface for an event bus that can publish and subscribe to messages.
type EventBus interface {
	// Publish publishes a message to the specified topic.
	// The payload will be marshaled into JSON.
	Publish(ctx context.Context, topic string, payload interface{}) error

	// Subscribe subscribes to a topic and returns a channel of messages.
	Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error)

	// Close closes the underlying resources held by the EventBus implementation
	// (e.g., publisher, subscriber connections).
	Close() error
}
