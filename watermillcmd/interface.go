package watermillcmd

import "context"

type CommandBus interface {
	Send(ctx context.Context, command interface{}) error
	// Add other methods from cqrs.CommandBus as needed
}

// EventBus defines the interface for an event bus.
type EventBus interface {
	Publish(ctx context.Context, events ...interface{}) error
	// Add other methods from your event bus implementation as needed
}
