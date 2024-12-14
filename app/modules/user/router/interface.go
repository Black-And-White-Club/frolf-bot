package userrouter

import (
	"context"

	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/db"
	"github.com/Black-And-White-Club/tcr-bot/internal/commands"
)

// CommandRouter interface for the user module
type CommandRouter interface {
	CreateUser(ctx context.Context, discordID string, name string, role userdb.UserRole, tagNumber int) error
	UpdateUser(ctx context.Context, discordID string, updates map[string]interface{}) error
}

// CommandBus is the interface for the command bus.
type CommandBus interface {
	Send(ctx context.Context, cmd commands.Command) error
}
