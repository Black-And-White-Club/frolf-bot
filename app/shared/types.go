package shared

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
)

// EventType represents a structured event type with a module and name.
type EventType struct {
	Module string
	Name   string
}

// String returns a string representation of the EventType.
func (e EventType) String() string {
	return e.Module + "." + e.Name
}

// MiddlewareFunc is a function type for middleware (if needed in the future).
type MiddlewareFunc func(ctx context.Context, eventType EventType, msg *message.Message) error
