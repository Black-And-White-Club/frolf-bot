package events

import (
	"context"
	"fmt"
	"strings"

	"github.com/Black-And-White-Club/tcr-bot/app/types"
)

// EventType represents a structured event type with a module and name.
type EventType struct {
	Module string
	Name   string
}

// MiddlewareFunc defines the signature for middleware functions.
type MiddlewareFunc func(ctx context.Context, eventType EventType, msg types.Message) error

// String returns a string representation of the EventType.
func (e EventType) String() string {
	return e.Module + "." + e.Name
}

// EventBus defines the interface for an event bus.
type EventBus interface {
	// Publish publishes an event to the event bus.
	Publish(ctx context.Context, eventType EventType, msg types.Message) error

	// PublishWithMetadata publishes an event with metadata.
	PublishWithMetadata(ctx context.Context, eventType EventType, msg types.Message, metadata map[string]string) error

	// Subscribe subscribes a handler function to a specific topic.
	Subscribe(ctx context.Context, topic string, handler func(ctx context.Context, msg types.Message) error) error

	// Start starts the event bus.
	Start(ctx context.Context) error

	// Stop stops the event bus.
	Stop(ctx context.Context) error

	// RegisterNotFoundHandler registers a handler for events with no subscriber.
	RegisterNotFoundHandler(handler func(eventType EventType) error)

	// RegisterMiddleware registers middleware to be executed before publishing.
	RegisterMiddleware(middleware MiddlewareFunc)
}

// ParseEventType converts a string topic to an EventType struct.
func ParseEventType(topic string) (EventType, error) {
	parts := strings.Split(topic, ".")
	if len(parts) != 2 {
		return EventType{}, fmt.Errorf("invalid topic format: %s", topic)
	}
	return EventType{Module: parts[0], Name: parts[1]}, nil
}

// ValidateEventType checks if an EventType is valid.
func ValidateEventType(eventType EventType) error {
	if eventType.Module == "" || eventType.Name == "" {
		return fmt.Errorf("invalid event type: %+v", eventType)
	}
	return nil
}
