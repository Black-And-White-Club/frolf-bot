package scoredb

import (
	"context"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/uptrace/bun"
)

// Repository defines the contract for score persistence.
// All methods are context-aware for cancellation and timeout propagation.
// Each method accepts an optional bun.IDB to support transaction sharing.
//
// Error semantics:
//   - ErrNotFound: Record does not exist (Get methods)
//   - ErrNoRowsAffected: UPDATE/DELETE matched no rows
//   - Other errors: Infrastructure failures (DB connection, query errors)
type Repository interface {
	// LogScores inserts or updates scores for a round.
	// Uses upsert pattern (ON CONFLICT DO UPDATE).
	LogScores(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, scores []sharedtypes.ScoreInfo, source string) error

	// UpdateScore updates a single user's score within an existing round record.
	// Returns ErrNotFound if the round record does not exist.
	UpdateScore(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, userID sharedtypes.DiscordID, newScore sharedtypes.Score) error

	// UpdateOrAddScore updates or adds a single score to a round record.
	// Contains self-healing logic for missing guild_id (data migration concern).
	// Creates a new round record if none exists.
	UpdateOrAddScore(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, scoreInfo sharedtypes.ScoreInfo) error

	// GetScoresForRound retrieves all scores for a round.
	// Returns nil if no scores exist (not an error condition).
	GetScoresForRound(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) ([]sharedtypes.ScoreInfo, error)
}
