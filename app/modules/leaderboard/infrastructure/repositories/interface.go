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
	// GetActiveLeaderboard retrieves the current active leaderboard for a guild.
	// Returns ErrNoActiveLeaderboard if no active leaderboard exists.
	GetActiveLeaderboard(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (*leaderboardtypes.Leaderboard, error)

	// SaveLeaderboard creates a new leaderboard version.
	// It deactivates any existing active leaderboard for the guild and inserts the new one.
	// This maintains the history of leaderboard states.
	SaveLeaderboard(ctx context.Context, db bun.IDB, leaderboard *leaderboardtypes.Leaderboard) error

	// DeactivateLeaderboard deactivates a specific leaderboard by ID.
	DeactivateLeaderboard(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, leaderboardID int64) error
}
