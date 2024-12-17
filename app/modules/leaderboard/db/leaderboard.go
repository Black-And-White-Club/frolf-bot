package leaderboarddb

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

// LeaderboardDBImpl implements the LeaderboardDBImpl interface using bun.
type LeaderboardDBImpl struct {
	DB *bun.DB
}

// GetLeaderboard retrieves the active leaderboard.
func (lb *LeaderboardDBImpl) GetLeaderboard(ctx context.Context) (*Leaderboard, error) {
	var leaderboard Leaderboard
	err := lb.DB.NewSelect().
		Model(&leaderboard).
		Where("active = ?", true).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch leaderboard: %w", err)
	}
	return &leaderboard, nil
}

// DeactivateCurrentLeaderboard deactivates the currently active leaderboard.
func (lb *LeaderboardDBImpl) DeactivateCurrentLeaderboard(ctx context.Context) error {
	_, err := lb.DB.NewUpdate().
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
func (lb *LeaderboardDBImpl) InsertLeaderboard(ctx context.Context, leaderboardData map[int]string, active bool) error {
	newLeaderboard := &Leaderboard{
		LeaderboardData: leaderboardData,
		Active:          active,
	}

	_, err := lb.DB.NewInsert().
		Model(newLeaderboard).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to insert new leaderboard: %w", err)
	}
	return nil
}

// UpdateLeaderboard updates the leaderboard within a transaction.
func (lb *LeaderboardDBImpl) UpdateLeaderboard(ctx context.Context, leaderboardData map[int]string) error { // Renamed function
	tx, err := lb.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Retrieve the active leaderboard
	var leaderboard Leaderboard
	err = tx.NewSelect().
		Model(&leaderboard).
		Where("active = ?", true).
		Scan(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch leaderboard: %w", err)
	}

	// Update the leaderboard data
	leaderboard.LeaderboardData = leaderboardData

	// Save the updated leaderboard
	_, err = tx.NewUpdate().
		Model(&leaderboard).
		Column("leaderboard_data").
		WherePK().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update leaderboard: %w", err)
	}

	return tx.Commit()
}

// SwapTags swaps the tags of two users in the leaderboard.
func (lb *LeaderboardDBImpl) SwapTags(ctx context.Context, requestorID, targetID string) error {
	// 1. Fetch the leaderboard data
	leaderboard, err := lb.GetLeaderboard(ctx)
	if err != nil {
		return fmt.Errorf("SwapTags: failed to get leaderboard: %w", err)
	}

	// 2. Find the tag numbers for the two users
	var requestorTag, targetTag int
	for tag, id := range leaderboard.LeaderboardData {
		if id == requestorID {
			requestorTag = tag
		} else if id == targetID {
			targetTag = tag
		}
	}

	// 3. If either user is not found, return an error
	if requestorTag == 0 || targetTag == 0 {
		return fmt.Errorf("SwapTags: one or both users not found on the leaderboard")
	}

	// 4. Swap the tags in the leaderboard data
	leaderboard.LeaderboardData[requestorTag], leaderboard.LeaderboardData[targetTag] = leaderboard.LeaderboardData[targetTag], leaderboard.LeaderboardData[requestorTag]

	// 5. Update the leaderboard in the database
	err = lb.UpdateLeaderboard(ctx, leaderboard.LeaderboardData)
	if err != nil {
		return fmt.Errorf("SwapTags: failed to update leaderboard: %w", err)
	}

	return nil
}

// InsertTagAndDiscordID inserts a new tag and Discord ID into the leaderboard.
func (lb *LeaderboardDBImpl) InsertTagAndDiscordID(ctx context.Context, tagNumber int, discordID string) error {
	// 1. Fetch the leaderboard data
	leaderboard, err := lb.GetLeaderboard(ctx)
	if err != nil {
		return fmt.Errorf("InsertTagAndDiscordID: failed to get leaderboard: %w", err)
	}

	// 2. Check if the tag is already taken
	if _, taken := leaderboard.LeaderboardData[tagNumber]; taken {
		return fmt.Errorf("InsertTagAndDiscordID: tag number %d is already taken", tagNumber)
	}

	// 3. Add the new tag and Discord ID to the leaderboard data
	leaderboard.LeaderboardData[tagNumber] = discordID

	// 4. Update the leaderboard in the database
	err = lb.UpdateLeaderboard(ctx, leaderboard.LeaderboardData)
	if err != nil {
		return fmt.Errorf("InsertTagAndDiscordID: failed to update leaderboard: %w", err)
	}

	return nil
}
