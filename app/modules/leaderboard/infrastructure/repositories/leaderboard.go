package leaderboarddb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/uptrace/bun"
)

// Impl implements the Repository interface using Bun ORM.
type Impl struct {
	db bun.IDB
}

// NewRepository creates a new leaderboard repository.
func NewRepository(db bun.IDB) Repository {
	return &Impl{db: db}
}

// --- READ METHODS ---

// GetActiveLeaderboard retrieves the current active leaderboard for a guild.
func (r *Impl) GetActiveLeaderboard(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (*leaderboardtypes.Leaderboard, error) {
	if db == nil {
		db = r.db
	}

	model := new(Leaderboard)
	err := db.NewSelect().
		Model(model).
		Where("is_active = ?", true).
		Where("guild_id = ?", guildID).
		Limit(1).
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoActiveLeaderboard
		}
		return nil, fmt.Errorf("leaderboarddb.GetActiveLeaderboard: %w", err)
	}
	return toSharedModel(model), nil
}

// --- WRITE METHODS ---

// SaveLeaderboard creates a new leaderboard version.
// It deactivates any existing active leaderboard for the guild and inserts the new one.
func (r *Impl) SaveLeaderboard(ctx context.Context, db bun.IDB, leaderboard *leaderboardtypes.Leaderboard) error {
	if db == nil {
		db = r.db
	}

	dbModel := toDBModel(leaderboard)
	dbModel.IsActive = true

	// 1. Deactivate the current leaderboard
	// We do this blindly to ensure only one is active.
	_, err := db.NewUpdate().
		Model((*Leaderboard)(nil)).
		Set("is_active = ?", false).
		Where("is_active = ?", true).
		Where("guild_id = ?", dbModel.GuildID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("leaderboarddb.SaveLeaderboard.deactivate: %w", err)
	}

	// 2. Insert the new leaderboard
	_, err = db.NewInsert().
		Model(dbModel).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("leaderboarddb.SaveLeaderboard.insert: %w", err)
	}

	// Update the shared model with the generated ID and UpdateID (if generated)
	leaderboard.ID = dbModel.ID
	leaderboard.UpdateID = dbModel.UpdateID

	return nil
}

// DeactivateLeaderboard deactivates a specific leaderboard by ID.
func (r *Impl) DeactivateLeaderboard(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, leaderboardID int64) error {
	if db == nil {
		db = r.db
	}
	_, err := db.NewUpdate().
		Model((*Leaderboard)(nil)).
		Set("is_active = ?", false).
		Where("id = ?", leaderboardID).
		Where("guild_id = ?", guildID).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("leaderboarddb.DeactivateLeaderboard: %w", err)
	}
	return nil
}

// =============================================================================
// Model Conversion Helpers
// =============================================================================

// toSharedModel converts the DB model to the shared domain type.
func toSharedModel(l *Leaderboard) *leaderboardtypes.Leaderboard {
	if l == nil {
		return nil
	}
	return &leaderboardtypes.Leaderboard{
		ID:              l.ID,
		LeaderboardData: l.LeaderboardData,
		IsActive:        l.IsActive,
		UpdateSource:    l.UpdateSource,
		UpdateID:        l.UpdateID,
		GuildID:         l.GuildID,
	}
}

// toDBModel converts the shared domain type to the DB model.
func toDBModel(l *leaderboardtypes.Leaderboard) *Leaderboard {
	if l == nil {
		return nil
	}
	return &Leaderboard{
		ID:              l.ID,
		LeaderboardData: l.LeaderboardData,
		IsActive:        l.IsActive,
		UpdateSource:    l.UpdateSource,
		UpdateID:        l.UpdateID,
		GuildID:         l.GuildID,
	}
}
