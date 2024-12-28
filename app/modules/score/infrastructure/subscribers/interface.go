package scoresubscribers

import (
	"context"
)

// ScoreSubscribers defines the interface for subscribing to score events.
type Subscribers interface {
	SubscribeToScoreEvents(ctx context.Context) error
}
