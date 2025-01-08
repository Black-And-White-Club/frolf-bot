// app/adapters/pub_adapter.go
package adapters

import (
	"context"
	"fmt"

	"github.com/Black-And-White-Club/tcr-bot/app/shared"
	"github.com/ThreeDotsLabs/watermill/message"
)

// WatermillPublisherAdapter adapts a Watermill publisher to the Publisher interface.
type WatermillPublisherAdapter struct {
	Publisher   message.Publisher
	middlewares []shared.MiddlewareFunc // Use a slice
}

// NewWatermillPublisherAdapter creates a new adapter for a Watermill publisher.
func NewWatermillPublisherAdapter(pub message.Publisher) shared.EventBus {
	return &WatermillPublisherAdapter{
		Publisher:   pub,
		middlewares: []shared.MiddlewareFunc{}, // Initialize the slice
	}
}

// Publish publishes messages to the given topic.
func (w *WatermillPublisherAdapter) Publish(ctx context.Context, eventType shared.EventType, msg shared.Message) error {
	// Apply middlewares
	for _, mw := range w.middlewares {
		if err := mw(ctx, eventType, &msg); err != nil {
			return fmt.Errorf("middleware error: %w", err)
		}
	}

	topic := eventType.String()
	return w.Publisher.Publish(topic, msg.(*WatermillMessageAdapter).Message)
}

// PublishWithMetadata publishes an event with metadata.
func (w *WatermillPublisherAdapter) PublishWithMetadata(ctx context.Context, eventType shared.EventType, msg shared.Message, metadata map[string]string) error {
	// Create a Watermill message
	wmMsg := msg.(*WatermillMessageAdapter).Message

	// Add metadata to the Watermill message
	for key, value := range metadata {
		wmMsg.Metadata[key] = value
	}

	topic := eventType.String()
	return w.Publisher.Publish(topic, wmMsg)
}

// These methods are required to implement shared.EventBus, but they are not used in this adapter.
func (w *WatermillPublisherAdapter) Subscribe(ctx context.Context, topic string, handler func(ctx context.Context, msg shared.Message) error) error {
	return nil
}
func (w *WatermillPublisherAdapter) RegisterNotFoundHandler(handler func(eventType shared.EventType) error) {
}
