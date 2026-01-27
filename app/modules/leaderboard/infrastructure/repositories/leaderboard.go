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
func (r *Impl) GetActiveLeaderboard(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (*Leaderboard, error) {
	// Fallback to default DB if nil is passed
	if db == nil {
		db = r.db
	}

	leaderboard := new(Leaderboard)
	err := db.NewSelect().
		Model(leaderboard).
		Column("id", "leaderboard_data", "is_active", "update_source", "update_id", "guild_id").
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
	return leaderboard, nil
}

// --- WRITE METHODS ---

func (r *Impl) CreateLeaderboard(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, leaderboard *Leaderboard) (*Leaderboard, error) {
	if db == nil {
		db = r.db
	}
	leaderboard.GuildID = guildID

	_, err := db.NewInsert().
		Model(leaderboard).
		Returning("*").
		Exec(ctx)

	if err != nil {
		return nil, fmt.Errorf("leaderboarddb.CreateLeaderboard: %w", err)
	}
	return leaderboard, nil
}

func (r *Impl) UpdateLeaderboard(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, leaderboardData leaderboardtypes.LeaderboardData, updateID sharedtypes.RoundID, source sharedtypes.ServiceUpdateSource) (*Leaderboard, error) {
	if db == nil {
		db = r.db
	}

	// 1. Deactivate the current leaderboard
	_, err := db.NewUpdate().
		Model((*Leaderboard)(nil)).
		Set("is_active = ?", false).
		Where("is_active = ?", true).
		Where("guild_id = ?", guildID).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("leaderboarddb.UpdateLeaderboard.deactivate: %w", err)
	}

	// 2. Create a new leaderboard
	newLeaderboard := &Leaderboard{
		LeaderboardData: leaderboardData,
		IsActive:        true,
		UpdateSource:    source,
		UpdateID:        updateID,
		GuildID:         guildID,
	}

	_, err = db.NewInsert().
		Model(newLeaderboard).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("leaderboarddb.UpdateLeaderboard.insert: %w", err)
	}

	return newLeaderboard, nil
}

// The Service can call this alone, or as part of a larger transaction.
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
	return err
}

// --- DOMAIN LOGIC ON MODELS ---

func (l *Leaderboard) HasTagNumber(tagNumber sharedtypes.TagNumber) bool {
	for _, entry := range l.LeaderboardData {
		if entry.TagNumber != 0 && entry.TagNumber == tagNumber {
			return true
		}
	}
	return false
}

func (l *Leaderboard) HasUserID(userID sharedtypes.DiscordID) bool {
	for _, entry := range l.LeaderboardData {
		if entry.UserID == userID {
			return true
		}
	}
	return false
}
