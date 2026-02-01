package userdb

import (
	"context"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/uptrace/bun"
)

// Repository defines the persistence contract for user data.
//
// Error semantics:
//   - ErrNotFound: requested record does not exist (Get* methods)
//   - ErrNoRowsAffected: UPDATE/DELETE matched no rows
//   - other errors: infrastructure failures
type Repository interface {
	// Global user operations
	GetUserGlobal(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID) (*User, error)
	GetByUserIDs(ctx context.Context, db bun.IDB, userIDs []sharedtypes.DiscordID) ([]*User, error)
	SaveGlobalUser(ctx context.Context, db bun.IDB, user *User) error
	UpdateGlobalUser(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, updates *UserUpdateFields) error

	// Guild membership operations
	CreateGuildMembership(ctx context.Context, db bun.IDB, membership *GuildMembership) error
	GetGuildMembership(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID) (*GuildMembership, error)
	UpdateMembershipRole(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID, role sharedtypes.UserRoleEnum) error
	GetUserMemberships(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID) ([]*GuildMembership, error)

	// Guild-scoped operations
	GetUserByUserID(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID) (*UserWithMembership, error)
	GetUserRole(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID) (sharedtypes.UserRoleEnum, error)
	UpdateUserRole(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID, role sharedtypes.UserRoleEnum) error
	FindByUDiscUsername(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, username string) (*UserWithMembership, error)
	FindByUDiscName(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, name string) (*UserWithMembership, error)
	FindByUDiscNameFuzzy(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, partialName string) ([]*UserWithMembership, error)

	// Profile operations
	UpdateProfile(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, displayName string, avatarHash string) error
}
