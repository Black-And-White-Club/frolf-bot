package userdb

import (
	"context"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// UserDB is an interface for interacting with the user database.
type UserDB interface {
	CreateUser(ctx context.Context, user *User) error
	GetUserByUserID(ctx context.Context, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID) (*User, error)
	GetUserRole(ctx context.Context, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID) (sharedtypes.UserRoleEnum, error)
	UpdateUserRole(ctx context.Context, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID, role sharedtypes.UserRoleEnum) error
}
