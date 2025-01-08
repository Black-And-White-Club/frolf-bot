// In shared/types.go (or a similar file in your shared package)
package shared

import "context"

// EventType represents a structured event type with a module and name.
type EventType struct {
	Module string
	Name   string
}
type MiddlewareFunc func(ctx context.Context, eventType EventType, msg *Message) error

// String returns a string representation of the EventType.
func (e EventType) String() string {
	return e.Module + "." + e.Name
}
