package leaderboarddb

import (
	"context"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/uptrace/bun"
)

// Repository defines the contract for leaderboard persistence.
// All methods are context-aware for cancellation and timeout propagation.
//
// Error semantics:
//   - ErrNotFound: Record does not exist
//   - ErrNoRowsAffected: UPDATE/DELETE matched no rows
//   - Other errors: Infrastructure failures (DB connection, query errors)
type Repository interface {
	// SavePointHistory records points earned by a member.
	SavePointHistory(ctx context.Context, db bun.IDB, guildID string, history *PointHistory) error

	// BulkSavePointHistory records multiple point history records efficiently.
	BulkSavePointHistory(ctx context.Context, db bun.IDB, guildID string, histories []*PointHistory) error

	// UpsertSeasonStanding updates or creates a season standing record.
	UpsertSeasonStanding(ctx context.Context, db bun.IDB, guildID string, standing *SeasonStanding) error

	// BulkUpsertSeasonStandings updates or creates multiple season standing records efficiently.
	BulkUpsertSeasonStandings(ctx context.Context, db bun.IDB, guildID string, standings []*SeasonStanding) error

	// GetSeasonStanding retrieves a member's season standing.
	GetSeasonStanding(ctx context.Context, db bun.IDB, guildID string, memberID sharedtypes.DiscordID) (*SeasonStanding, error)

	// GetSeasonBestTags retrieves the best tag for a list of members for a season.
	// If seasonID is empty, the active season is used.
	// Returns a map of MemberID -> BestTag.
	GetSeasonBestTags(ctx context.Context, db bun.IDB, guildID string, seasonID string, memberIDs []sharedtypes.DiscordID) (map[sharedtypes.DiscordID]int, error)

	// CountSeasonMembers returns the total number of members with a standing in a season (for tier calc).
	// If seasonID is empty, the active season is used.
	CountSeasonMembers(ctx context.Context, db bun.IDB, guildID string, seasonID string) (int, error)

	// GetSeasonStandings retrieves season standings for a batch of members in a season.
	// If seasonID is empty, the active season is used.
	// Returns a map of MemberID -> *SeasonStanding.
	GetSeasonStandings(ctx context.Context, db bun.IDB, guildID string, seasonID string, memberIDs []sharedtypes.DiscordID) (map[sharedtypes.DiscordID]*SeasonStanding, error)

	// GetPointHistoryForRound retrieves all point history records for a specific round.
	GetPointHistoryForRound(ctx context.Context, db bun.IDB, guildID string, roundID sharedtypes.RoundID) ([]PointHistory, error)

	// DeletePointHistoryForRound deletes all point history records for a specific round.
	DeletePointHistoryForRound(ctx context.Context, db bun.IDB, guildID string, roundID sharedtypes.RoundID) error

	// DecrementSeasonStanding decrements a member's season standing points and rounds played.
	// If seasonID is empty, the active season is used.
	DecrementSeasonStanding(ctx context.Context, db bun.IDB, guildID string, memberID sharedtypes.DiscordID, seasonID string, pointsToRemove int) error

	// --- Season Management ---

	// GetActiveSeason retrieves the currently active season.
	GetActiveSeason(ctx context.Context, db bun.IDB, guildID string) (*Season, error)

	// CreateSeason creates a new season record.
	CreateSeason(ctx context.Context, db bun.IDB, guildID string, season *Season) error

	// DeactivateAllSeasons sets is_active=false for all seasons.
	DeactivateAllSeasons(ctx context.Context, db bun.IDB, guildID string) error

	// GetPointHistoryForMember retrieves point history for a member, ordered by created_at desc.
	GetPointHistoryForMember(ctx context.Context, db bun.IDB, guildID string, memberID sharedtypes.DiscordID, limit int) ([]PointHistory, error)

	// GetSeasonStandingsBySeasonID retrieves all standings for a specific season.
	GetSeasonStandingsBySeasonID(ctx context.Context, db bun.IDB, guildID string, seasonID string) ([]SeasonStanding, error)

	// ListSeasons retrieves all seasons for a guild, ordered by is_active DESC, start_date DESC.
	ListSeasons(ctx context.Context, db bun.IDB, guildID string) ([]Season, error)

	// GetSeasonByID retrieves a single season by its ID.
	GetSeasonByID(ctx context.Context, db bun.IDB, guildID string, seasonID string) (*Season, error)
}
