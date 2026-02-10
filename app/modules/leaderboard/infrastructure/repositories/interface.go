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

	// SavePointHistory records points earned by a member.
	SavePointHistory(ctx context.Context, db bun.IDB, history *PointHistory) error

	// UpsertSeasonStanding updates or creates a season standing record.
	UpsertSeasonStanding(ctx context.Context, db bun.IDB, standing *SeasonStanding) error

	// GetSeasonStanding retrieves a member's season standing.
	GetSeasonStanding(ctx context.Context, db bun.IDB, memberID sharedtypes.DiscordID) (*SeasonStanding, error)

	// GetSeasonBestTags retrieves the best tag for a list of members for the current season.
	// Returns a map of MemberID -> BestTag.
	GetSeasonBestTags(ctx context.Context, db bun.IDB, memberIDs []sharedtypes.DiscordID) (map[sharedtypes.DiscordID]int, error)

	// CountSeasonMembers returns the total number of members with a standing in the current season (for tier calc).
	CountSeasonMembers(ctx context.Context, db bun.IDB) (int, error)

	// GetSeasonStandings retrieves season standings for a batch of members.
	// Returns a map of MemberID -> *SeasonStanding.
	GetSeasonStandings(ctx context.Context, db bun.IDB, memberIDs []sharedtypes.DiscordID) (map[sharedtypes.DiscordID]*SeasonStanding, error)

	// GetPointHistoryForRound retrieves all point history records for a specific round.
	GetPointHistoryForRound(ctx context.Context, db bun.IDB, roundID sharedtypes.RoundID) ([]PointHistory, error)

	// DeletePointHistoryForRound deletes all point history records for a specific round.
	DeletePointHistoryForRound(ctx context.Context, db bun.IDB, roundID sharedtypes.RoundID) error

	// DecrementSeasonStanding decrements a member's season standing points and rounds played.
	DecrementSeasonStanding(ctx context.Context, db bun.IDB, memberID sharedtypes.DiscordID, pointsToRemove int) error
}
