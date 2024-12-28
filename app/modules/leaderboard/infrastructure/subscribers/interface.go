package leaderboardsubscribers

import (
	"context"
)

// Subscribers defines the interface for subscribing to leaderboard events.
type Subscribers interface {
	SubscribeToLeaderboardEvents(ctx context.Context) error
}
