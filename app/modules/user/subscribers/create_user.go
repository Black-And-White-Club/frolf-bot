package usercommands

import (
	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/db"
	"github.com/Black-And-White-Club/tcr-bot/internal/commands"
)

// CreateUserRequest represents the request to create a user.
type CreateUserRequest struct {
	DiscordID string          `json:"discord_id"`
	Name      string          `json:"name"`
	Role      userdb.UserRole `json:"role"`
	TagNumber int             `json:"tag_number"`
}

// CommandName returns the command name for CreateUserRequest
func (cmd CreateUserRequest) CommandName() string {
	return "create_user"
}

var _ commands.Command = CreateUserRequest{}
