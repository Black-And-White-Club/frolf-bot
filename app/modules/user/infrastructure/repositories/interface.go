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
	FindByUDiscNameFuzzy(ctx context.Context, guildID sharedtypes.GuildID, partialName string) ([]*UserWithMembership, error)
}

// Backwards-compatibility aliases for callers that used the old naming.
type (
	UserDB = Repository
)

// NewUserDB is kept for compatibility; use NewRepository when possible.
// Implementations should provide NewRepository.
// NewUserDB is an alias for NewRepository to preserve existing callers.
func NewUserDB(db bun.IDB) Repository { return NewRepository(db) }
