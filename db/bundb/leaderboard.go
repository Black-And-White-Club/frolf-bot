package bundb

import (
	"context"
	"fmt"

	"github.com/Black-And-White-Club/tcr-bot/models"
	"github.com/uptrace/bun"
)

// leaderboardDB implements the LeaderboardDB interface using bun.
type leaderboardDB struct {
	db *bun.DB
}

// GetLeaderboard retrieves the active leaderboard.
func (lb *leaderboardDB) GetLeaderboard(ctx context.Context) (*models.Leaderboard, error) {
	var leaderboard models.Leaderboard
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
func (lb *leaderboardDB) GetLeaderboardTagData(ctx context.Context) (*models.Leaderboard, error) {
	var leaderboard models.Leaderboard
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
func (lb *leaderboardDB) DeactivateCurrentLeaderboard(ctx context.Context) error {
	_, err := lb.db.NewUpdate().
		Model((*models.Leaderboard)(nil)).
		Set("active = ?", false).
		Where("active = ?", true).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to deactivate current leaderboard: %w", err)
	}
	return nil
}

// InsertLeaderboard inserts a new leaderboard into the database.
func (lb *leaderboardDB) InsertLeaderboard(ctx context.Context, leaderboardData map[int]string, active bool) error {
	newLeaderboard := &models.Leaderboard{
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
func (lb *leaderboardDB) UpdateLeaderboardWithTransaction(ctx context.Context, leaderboardData map[int]string) error {
	// 1. Begin a transaction
	tx, err := lb.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// 2. Fetch the active leaderboard within the transaction
	leaderboard, err := lb.GetLeaderboard(ctx) // Fetch the leaderboard
	if err != nil {
		return fmt.Errorf("failed to get leaderboard: %w", err)
	}

	// 3. Deactivate the current leaderboard within the transaction
	if err := lb.DeactivateCurrentLeaderboard(ctx); err != nil {
		return err
	}

	// 4. Insert the new leaderboard within the transaction
	if err := lb.InsertLeaderboard(ctx, leaderboard.LeaderboardData, true); err != nil { // Pass leaderboardData and active status
		return err
	}

	// 5. Commit the transaction if everything was successful
	return tx.Commit()
}
