package userdb

import (
	"context"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// UserDB is an interface for interacting with the user database.
type UserDB interface {
	// Global user operations
	GetUserGlobal(ctx context.Context, userID sharedtypes.DiscordID) (*User, error)
	CreateGlobalUser(ctx context.Context, user *User) error
	UpdateUDiscIdentityGlobal(ctx context.Context, userID sharedtypes.DiscordID, username *string, name *string) error

	// Guild membership operations
	CreateGuildMembership(ctx context.Context, membership *GuildMembership) error
	GetGuildMembership(ctx context.Context, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID) (*GuildMembership, error)
	UpdateMembershipRole(ctx context.Context, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID, role sharedtypes.UserRoleEnum) error
	GetUserMemberships(ctx context.Context, userID sharedtypes.DiscordID) ([]*GuildMembership, error)

	// Guild-scoped operations
	GetUserByUserID(ctx context.Context, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID) (*UserWithMembership, error)
	GetUserRole(ctx context.Context, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID) (sharedtypes.UserRoleEnum, error)
	UpdateUserRole(ctx context.Context, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID, role sharedtypes.UserRoleEnum) error
	FindByUDiscUsername(ctx context.Context, guildID sharedtypes.GuildID, username string) (*UserWithMembership, error)
	FindByUDiscName(ctx context.Context, guildID sharedtypes.GuildID, name string) (*UserWithMembership, error)
}
