package usercommands

import (
	"context"

	"github.com/Black-And-White-Club/tcr-bot/watermillcmd"
)

// CommandService defines the interface for user commands.
type UserService interface {
	CreateUser(ctx context.Context, discordID string, name string, role string, tagNumber int) error
	UpdateUser(ctx context.Context, discordID string, updates map[string]interface{}) error
	// Add other command methods as needed (e.g., DeleteUser)

	CommandBus() watermillcmd.CommandBus
}
