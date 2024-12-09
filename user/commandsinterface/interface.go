// user/commandsinterface/interface.go
package usercmdsinterface

import (
	"context"

	"github.com/Black-And-White-Club/tcr-bot/user/db"
)

// CommandService defines the interface for user commands.
type CommandService interface {
	CreateUser(ctx context.Context, user *db.User, tagNumber int) error
	UpdateUser(ctx context.Context, discordID string, updates *db.User) error
	// Add other command methods as needed (e.g., DeleteUser)
}
