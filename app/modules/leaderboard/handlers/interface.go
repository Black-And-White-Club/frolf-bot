package leaderboardhandlers

import (
	"context"

	"github.com/Black-And-White-Club/tcr-bot/internal/commands"
)

// LeaderboardCommandHandler is an interface for handlers that process user commands.
type LeaderboardCommandHandler interface {
	Handle(ctx context.Context, cmd commands.Command) error
}

// UserQueryHandler is an interface for handlers that process user queries.
type LeaderboardQueryHandler interface {
	// Handle(ctx context.Context, query interface{}) (interface{}, error)
}
