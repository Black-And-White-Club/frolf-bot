package userdb

import (
	"context"

	usertypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types"
)

// UserDB is an interface for interacting with the user database.
type UserDB interface {
	CreateUser(ctx context.Context, user usertypes.User) error
	GetUserByDiscordID(ctx context.Context, discordID usertypes.DiscordID) (usertypes.User, error)
	GetUserRole(ctx context.Context, discordID usertypes.DiscordID) (usertypes.UserRoleEnum, error)
	UpdateUserRole(ctx context.Context, discordID usertypes.DiscordID, newRole usertypes.UserRoleEnum) error
}
