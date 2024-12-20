package roundsubscribers

import (
	"context"
)

// RoundSubscribers defines the interface for subscribing to round events.
type Subscribers interface {
	SubscribeToRoundManagementEvents(ctx context.Context) error
	SubscribeToParticipantManagementEvents(ctx context.Context) error
	SubscribeToRoundFinalizationEvents(ctx context.Context) error
	SubscribeToRoundStartedEvents(ctx context.Context) error
}
