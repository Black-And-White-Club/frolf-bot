package scoresubscribers

import (
	"context"
)

// ScoreEventSubscriber defines the interface for subscribing to score events.
type Subscribers interface {
	SubscribeToScoreEvents(ctx context.Context) error
}
