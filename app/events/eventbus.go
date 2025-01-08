package events

import (
	"context"
	"fmt"
	"strings"

	"github.com/Black-And-White-Club/tcr-bot/app/adapters"
	"github.com/Black-And-White-Club/tcr-bot/app/shared"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// eventBus implements the shared.EventBus interface.
type eventBus struct {
	adapter         shared.EventAdapterInterface
	middlewares     []shared.MiddlewareFunc
	notFoundHandler func(eventType shared.EventType) error
}

// NewEventBus creates and returns an EventBus with the given EventAdapterInterface.
func NewEventBus(adapter shared.EventAdapterInterface) shared.EventBus {
	return &eventBus{
		adapter:     adapter,
		middlewares: []shared.MiddlewareFunc{},
	}
}

// Publish publishes an event to the event bus.
func (eb *eventBus) Publish(ctx context.Context, eventType shared.EventType, payload []byte, metadata map[string]string) error {
	// Validate the event type
	if err := ValidateEventType(eventType); err != nil {
		return fmt.Errorf("invalid event type: %w", err)
	}

	// Create a new Watermill message
	msg := message.NewMessage(watermill.NewUUID(), payload)

	// Apply metadata
	for k, v := range metadata {
		msg.Metadata.Set(k, v)
	}

	// Apply middlewares (if any)
	// ... (Adapt middleware to work with *message.Message) ...

	// Use the adapter to publish the message
	return eb.adapter.Publish(ctx, eventType, msg)
}

// Subscribe subscribes to a specific event type and processes messages with the given handler.
func (eb *eventBus) Subscribe(ctx context.Context, topic string, handler func(msg *message.Message) error) error {
	eventType, err := ParseEventType(topic)
	if err != nil {
		return fmt.Errorf("failed to parse topic: %w", err)
	}

	// Determine queue group based on your logic, or leave it empty for default behavior
	queueGroup := "" // You can modify this based on your requirements

	if err := eb.adapter.Subscribe(ctx, eventType, queueGroup, handler); err != nil {
		if eb.notFoundHandler != nil {
			return eb.notFoundHandler(eventType)
		}
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	return nil
}

// RegisterNotFoundHandler registers a handler for events with no subscriber.
func (eb *eventBus) RegisterNotFoundHandler(handler func(eventType shared.EventType) error) {
	eb.notFoundHandler = handler
}

// RegisterMiddleware registers middleware to be executed before publishing.
func (eb *eventBus) RegisterMiddleware(middleware shared.MiddlewareFunc) {
	eb.middlewares = append(eb.middlewares, middleware)
}

// ParseEventType converts a string topic to an EventType struct.
func ParseEventType(topic string) (shared.EventType, error) {
	parts := strings.Split(topic, ".")
	if len(parts) != 2 {
		return shared.EventType{}, fmt.Errorf("invalid topic format: %s", topic)
	}
	return shared.EventType{Module: parts[0], Name: parts[1]}, nil
}

// ValidateEventType checks if an EventType is valid.
func ValidateEventType(eventType shared.EventType) error {
	if eventType.Module == "" || eventType.Name == "" {
		return fmt.Errorf("invalid event type: %+v", eventType)
	}
	return nil
}

// NewStreamNamer creates a StreamNamingStrategy function that
// generates NATS stream names based on the event type.
func NewStreamNamer() adapters.StreamNamingStrategy {
	return func(eventType shared.EventType) string {
		return eventType.String()
	}
}
