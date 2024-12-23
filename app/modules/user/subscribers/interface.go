package usersubscribers

import "context"

// UserEventSubscriber defines the interface for subscribing to user events.
type UserEventSubscriber interface {
	SubscribeToUserEvents(ctx context.Context) error
}

// Closer interface for resources that need to be closed.
type Closer interface {
	Close() error
}
