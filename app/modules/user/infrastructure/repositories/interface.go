package userdb

import (
	"context"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Repository defines the persistence contract for user data.
//
// Error semantics:
//   - ErrNotFound: requested record does not exist (Get* methods)
//   - ErrNoRowsAffected: UPDATE/DELETE matched no rows
//   - other errors: infrastructure failures
type Repository interface {
	// ... (existing methods remain, but we add UUID resolution)
	GetUUIDByDiscordID(ctx context.Context, db bun.IDB, discordID sharedtypes.DiscordID) (uuid.UUID, error)
	GetClubUUIDByDiscordGuildID(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (uuid.UUID, error)
	GetDiscordGuildIDByClubUUID(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) (sharedtypes.GuildID, error)

	// Global user operations
	GetUserGlobal(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID) (*User, error)
	GetUserByUUID(ctx context.Context, db bun.IDB, uuid uuid.UUID) (*User, error)
	GetByUserIDs(ctx context.Context, db bun.IDB, userIDs []sharedtypes.DiscordID) ([]*User, error)
	SaveGlobalUser(ctx context.Context, db bun.IDB, user *User) error
	UpdateGlobalUser(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, updates *UserUpdateFields) error

	// Guild membership operations (Legacy Discord-scoped)
	CreateGuildMembership(ctx context.Context, db bun.IDB, membership *GuildMembership) error
	GetGuildMembership(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID) (*GuildMembership, error)
	UpdateMembershipRole(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID, role sharedtypes.UserRoleEnum) error
	GetUserMemberships(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID) ([]*GuildMembership, error)

	// Club operations (New Identity Abstraction)
	GetClubMembership(ctx context.Context, db bun.IDB, userUUID, clubUUID uuid.UUID) (*ClubMembership, error)
	GetClubMembershipsByUserUUID(ctx context.Context, db bun.IDB, userUUID uuid.UUID) ([]*ClubMembership, error)
	GetClubMembershipsByUserUUIDs(ctx context.Context, db bun.IDB, userUUIDs []uuid.UUID) ([]*ClubMembership, error)
	UpsertClubMembership(ctx context.Context, db bun.IDB, membership *ClubMembership) error
	GetClubMembershipByExternalID(ctx context.Context, db bun.IDB, externalID string, clubUUID uuid.UUID) (*ClubMembership, error)

	// Guild-scoped operations
	GetUserByUserID(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID) (*UserWithMembership, error)
	GetUserRole(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID) (sharedtypes.UserRoleEnum, error)
	UpdateUserRole(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, guildID sharedtypes.GuildID, role sharedtypes.UserRoleEnum) error
	FindByUDiscUsername(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, username string) (*UserWithMembership, error)
	FindGlobalByUDiscUsername(ctx context.Context, db bun.IDB, username string) (*User, error)
	FindByUDiscName(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, name string) (*UserWithMembership, error)
	GetUsersByUDiscNames(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, names []string) ([]UserWithMembership, error)
	GetUsersByUDiscUsernames(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, usernames []string) ([]UserWithMembership, error)
	FindByUDiscNameFuzzy(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, partialName string) ([]*UserWithMembership, error)

	// Profile operations
	UpdateProfile(ctx context.Context, db bun.IDB, userID sharedtypes.DiscordID, displayName string, avatarHash string) error

	// Refresh Token operations
	SaveRefreshToken(ctx context.Context, db bun.IDB, token *RefreshToken) error
	GetRefreshToken(ctx context.Context, db bun.IDB, hash string) (*RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, db bun.IDB, hash string) error
	RevokeAllUserTokens(ctx context.Context, db bun.IDB, userUUID uuid.UUID) error

	// Magic Link operations
	SaveMagicLink(ctx context.Context, db bun.IDB, link *MagicLink) error
	GetMagicLink(ctx context.Context, db bun.IDB, token string) (*MagicLink, error)
	MarkMagicLinkUsed(ctx context.Context, db bun.IDB, token string) error
}
