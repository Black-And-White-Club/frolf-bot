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

// --- READ METHODS (transaction-aware) ---

func (r *Impl) GetActiveLeaderboard(ctx context.Context, guildID sharedtypes.GuildID) (*Leaderboard, error) {
	return r.GetActiveLeaderboardIDB(ctx, r.db, guildID)
}

// GetActiveLeaderboardIDB is the transaction-aware version of the getter
func (r *Impl) GetActiveLeaderboardIDB(ctx context.Context, idb bun.IDB, guildID sharedtypes.GuildID) (*Leaderboard, error) {
	leaderboard := new(Leaderboard)
	err := idb.NewSelect().
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
		return nil, fmt.Errorf("leaderboarddb.GetActiveLeaderboardIDB: %w", err)
	}
	return leaderboard, nil
}

// --- WRITE METHODS (Now Transaction-Aware via bun.IDB) ---

// CreateLeaderboard inserts a new leaderboard record.
// It accepts bun.IDB so it can participate in the Service's transactions.
func (r *Impl) CreateLeaderboard(ctx context.Context, idb bun.IDB, guildID sharedtypes.GuildID, leaderboard *Leaderboard) (*Leaderboard, error) {
	leaderboard.GuildID = guildID

	// Ensure we use the passed 'idb' (which could be a transaction)
	_, err := idb.NewInsert().
		Model(leaderboard).
		Returning("*"). // Returning the whole model ensures ID and Timestamps are synced
		Exec(ctx)

	if err != nil {
		return nil, fmt.Errorf("leaderboarddb.CreateLeaderboard: %w", err)
	}

	return leaderboard, nil
}

// DeactivateLeaderboard deactivates the specified leaderboard using the provided IDB (DB or Tx).
func (r *Impl) DeactivateLeaderboard(ctx context.Context, idb bun.IDB, guildID sharedtypes.GuildID, leaderboardID int64) error {
	_, err := idb.NewUpdate().
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

// UpdateLeaderboard now accepts bun.IDB. It performs the "Deactivate Old -> Insert New" logic.
// This allows the Service to wrap this call and a "Publish to Outbox" call in one transaction.
func (r *Impl) UpdateLeaderboard(ctx context.Context, idb bun.IDB, guildID sharedtypes.GuildID, leaderboardData leaderboardtypes.LeaderboardData, updateID sharedtypes.RoundID, source sharedtypes.ServiceUpdateSource) (*Leaderboard, error) {
	// 1. Deactivate the current leaderboard
	_, err := idb.NewUpdate().
		Model((*Leaderboard)(nil)).
		Set("is_active = ?", false).
		Where("is_active = ?", true).
		Where("guild_id = ?", guildID).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("leaderboarddb.UpdateLeaderboard.deactivate: %w", err)
	}

	// 2. Create a new leaderboard with the updated data
	newLeaderboard := &Leaderboard{
		LeaderboardData: leaderboardData,
		IsActive:        true,
		UpdateSource:    source,
		UpdateID:        updateID,
		GuildID:         guildID,
	}

	_, err = idb.NewInsert().
		Model(newLeaderboard).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("leaderboarddb.UpdateLeaderboard.insert: %w", err)
	}

	return newLeaderboard, nil
}

// CheckTagAvailability remains similar but uses the IDB-aware getter
func (r *Impl) CheckTagAvailability(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, tagNumber sharedtypes.TagNumber) (TagAvailabilityResult, error) {
	leaderboard, err := r.GetActiveLeaderboardIDB(ctx, r.db, guildID)
	if err != nil {
		if errors.Is(err, ErrNoActiveLeaderboard) {
			// Flow for first-time guild setup
			return TagAvailabilityResult{Available: true}, nil
		}
		return TagAvailabilityResult{Available: false}, err
	}

	if leaderboard.HasUserID(userID) {
		return TagAvailabilityResult{Available: false, Reason: "user already has a tag"}, nil
	}
	if leaderboard.HasTagNumber(tagNumber) {
		return TagAvailabilityResult{Available: false, Reason: "tag already taken"}, nil
	}

	return TagAvailabilityResult{Available: true}, nil
}

// --- REFACTORED DOMAIN METHODS (Optional helpers) ---

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

// GetTagByUserID updated to use IDB-aware logic
func (r *Impl) GetTagByUserID(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (*sharedtypes.TagNumber, error) {
	activeLeaderboard, err := r.GetActiveLeaderboardIDB(ctx, r.db, guildID)
	if err != nil {
		return nil, err
	}

	for _, entry := range activeLeaderboard.LeaderboardData {
		if entry.UserID == userID && entry.TagNumber != 0 {
			tagVal := entry.TagNumber
			return &tagVal, nil
		}
	}

	return nil, sql.ErrNoRows
}
