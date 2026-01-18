package leaderboarddb

import (
	"context"

	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/uptrace/bun"
)

// TagAvailabilityResult represents the detailed result of a tag availability check.
type TagAvailabilityResult struct {
	Available bool
	Reason    string
}

// Repository defines the contract for leaderboard persistence.
// All methods are context-aware for cancellation and timeout propagation.
//
// Error semantics:
//   - ErrNotFound: Record does not exist
//   - ErrNoActiveLeaderboard: No active leaderboard for guild
//   - ErrUserTagNotFound: User has no tag in active leaderboard
//   - ErrNoRowsAffected: UPDATE/DELETE matched no rows
//   - Other errors: Infrastructure failures (DB connection, query errors)
type Repository interface {
	// Read Methods
	GetActiveLeaderboard(ctx context.Context, guildID sharedtypes.GuildID) (*Leaderboard, error)
	GetActiveLeaderboardIDB(ctx context.Context, idb bun.IDB, guildID sharedtypes.GuildID) (*Leaderboard, error)
	GetTagByUserID(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (*sharedtypes.TagNumber, error)
	CheckTagAvailability(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, tagNumber sharedtypes.TagNumber) (TagAvailabilityResult, error)

	// Write Methods (Atomic & Batch-Oriented)
	// We pass bun.IDB so the service can control the transaction.
	UpdateLeaderboard(ctx context.Context, idb bun.IDB, guildID sharedtypes.GuildID, leaderboardData leaderboardtypes.LeaderboardData, updateID sharedtypes.RoundID, source sharedtypes.ServiceUpdateSource) (*Leaderboard, error)
	CreateLeaderboard(ctx context.Context, idb bun.IDB, guildID sharedtypes.GuildID, leaderboard *Leaderboard) (*Leaderboard, error)
	DeactivateLeaderboard(ctx context.Context, idb bun.IDB, guildID sharedtypes.GuildID, leaderboardID int64) error
}

// Type aliases for backward compatibility during migration.
// Remove these after all callers are updated.
type (
	LeaderboardDB     = Repository
	LeaderboardDBImpl = Impl
)
