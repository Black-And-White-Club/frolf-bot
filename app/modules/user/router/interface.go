package userrouter

import (
	"context"

	usercommands "github.com/Black-And-White-Club/tcr-bot/app/modules/user/commands"
	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/db"
)

// CommandRouter interface for the user module
type CommandRouter interface {
	CreateUser(ctx context.Context, discordID string, name string, role userdb.UserRole, tagNumber int) error
	UpdateUser(ctx context.Context, discordID string, updates map[string]interface{}) error
}

// CommandBus is the interface for the command bus.
type CommandBus interface {
	Send(ctx context.Context, cmd usercommands.Command) error
}
