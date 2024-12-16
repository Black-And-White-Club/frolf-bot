package leaderboarddb

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

// leaderboardDBImpl implements the LeaderboardDBImpl interface using bun.
type leaderboardDBImpl struct {
	db *bun.DB
}

// GetLeaderboard retrieves the active leaderboard.
func (lb *leaderboardDBImpl) GetLeaderboard(ctx context.Context) (*Leaderboard, error) {
	var leaderboard Leaderboard
	err := lb.db.NewSelect().
		Model(&leaderboard).
		Where("active = ?", true).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch leaderboard: %w", err)
	}
	return &leaderboard, nil
}

// GetLeaderboardTagData retrieves the tag and Discord ID data for the active leaderboard.
func (lb *leaderboardDBImpl) GetLeaderboardTagData(ctx context.Context) (*Leaderboard, error) {
	var leaderboard Leaderboard
	err := lb.db.NewSelect().
		Model(&leaderboard).
		Column("leaderboard_data"). // Select only the leaderboard_data column
		Where("active = ?", true).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch leaderboard tag data: %w", err)
	}

	return &leaderboard, nil
}

// DeactivateCurrentLeaderboard deactivates the currently active leaderboard.
func (lb *leaderboardDBImpl) DeactivateCurrentLeaderboard(ctx context.Context) error {
	_, err := lb.db.NewUpdate().
		Model((*Leaderboard)(nil)).
		Set("active = ?", false).
		Where("active = ?", true).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to deactivate current leaderboard: %w", err)
	}
	return nil
}

// InsertLeaderboard inserts a new leaderboard into the database.
func (lb *leaderboardDBImpl) InsertLeaderboard(ctx context.Context, leaderboardData map[int]string, active bool) error {
	newLeaderboard := &Leaderboard{
		LeaderboardData: leaderboardData,
		Active:          active,
	}

	_, err := lb.db.NewInsert().
		Model(newLeaderboard).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to insert new leaderboard: %w", err)
	}
	return nil
}

// UpdateLeaderboardWithTransaction updates the leaderboard within a transaction.
func (lb *leaderboardDBImpl) UpdateLeaderboardWithTransaction(ctx context.Context, leaderboardData map[int]string) error {
	tx, err := lb.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Deactivate the current leaderboard
	_, err = tx.NewUpdate().
		Model((*Leaderboard)(nil)).
		Set("active = ?", false).
		Where("active = ?", true).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to deactivate current leaderboard: %w", err)
	}

	// Insert the new leaderboard
	newLeaderboard := &Leaderboard{
		LeaderboardData: leaderboardData,
		Active:          true,
	}
	_, err = tx.NewInsert().
		Model(newLeaderboard).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to insert new leaderboard: %w", err)
	}

	return tx.Commit()
}
