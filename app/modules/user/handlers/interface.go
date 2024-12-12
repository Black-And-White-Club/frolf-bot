package userhandlers

import (
	"context"

	usercommands "github.com/Black-And-White-Club/tcr-bot/app/modules/user/commands"
)

// UserCommandHandler is an interface for handlers that process user commands.
type UserCommandHandler interface {
	Handle(ctx context.Context, cmd usercommands.Command) error
}

// UserQueryHandler is an interface for handlers that process user queries.
type UserQueryHandler interface {
	// Handle(ctx context.Context, query interface{}) (interface{}, error)
}
