// In shared/eventbus.go (or a similar file in your shared package)
package shared

import (
	"context"
)

// EventBus defines the interface for an event bus.
type EventBus interface {

	// Publish publishes an event to the event bus.
	Publish(ctx context.Context, eventType EventType, msg Message) error

	// PublishWithMetadata publishes an event with metadata.
	PublishWithMetadata(ctx context.Context, eventType EventType, msg Message, metadata map[string]string) error

	// Subscribe subscribes a handler function to a specific topic.
	Subscribe(ctx context.Context, topic string, handler func(ctx context.Context, msg Message) error) error

	// RegisterNotFoundHandler registers a handler for events with no subscriber.
	RegisterNotFoundHandler(handler func(eventType EventType) error)
}
