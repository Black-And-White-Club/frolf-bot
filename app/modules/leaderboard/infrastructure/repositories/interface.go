package leaderboarddb

import (
	"context"

	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/uptrace/bun"
)

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
	GetActiveLeaderboard(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (*Leaderboard, error)

	// Write Methods (Atomic & Batch-Oriented)
	// We pass bun.IDB so the service can control the transaction.
	UpdateLeaderboard(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, leaderboardData leaderboardtypes.LeaderboardData, updateID sharedtypes.RoundID, source sharedtypes.ServiceUpdateSource) (*Leaderboard, error)
	CreateLeaderboard(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, leaderboard *Leaderboard) (*Leaderboard, error)
	DeactivateLeaderboard(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, leaderboardID int64) error
}
