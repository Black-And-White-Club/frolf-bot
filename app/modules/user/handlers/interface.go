package userhandlers

import (
	"context"

	"github.com/Black-And-White-Club/tcr-bot/internal/commands"
)

// UserCommandHandler is an interface for handlers that process user commands.
type UserCommandHandler interface {
	Handle(ctx context.Context, cmd commands.Command) error
}

// UserQueryHandler is an interface for handlers that process user queries.
type UserQueryHandler interface {
	// Handle(ctx context.Context, query interface{}) (interface{}, error)
}
