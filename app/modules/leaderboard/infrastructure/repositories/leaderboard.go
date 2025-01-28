package leaderboarddb

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/domain/types"
	"github.com/uptrace/bun"
)

// LeaderboardRepository handles database operations for leaderboards.
type LeaderboardDBImpl struct {
	DB *bun.DB
}

// GetActiveLeaderboard retrieves the currently active leaderboard.
func (db *LeaderboardDBImpl) GetActiveLeaderboard(ctx context.Context) (*Leaderboard, error) {
	leaderboard := new(Leaderboard)
	err := db.DB.NewSelect().Model(leaderboard).Where("is_active = ?", true).Scan(ctx)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no active leaderboard found")
		}
		return nil, fmt.Errorf("failed to get active leaderboard: %w", err)
	}
	return leaderboard, nil
}

// CreateLeaderboard creates a new leaderboard entry and returns its ID.
func (db *LeaderboardDBImpl) CreateLeaderboard(ctx context.Context, leaderboard *Leaderboard) (int64, error) {
	result, err := db.DB.NewInsert().Model(leaderboard).Exec(ctx)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get ID of newly created leaderboard: %w", err)
	}

	return id, nil
}

// DeactivateLeaderboard deactivates the specified leaderboard.
func (db *LeaderboardDBImpl) DeactivateLeaderboard(ctx context.Context, leaderboardID int64) error {
	_, err := db.DB.NewUpdate().
		Model((*Leaderboard)(nil)).
		Set("is_active = ?", false).
		Where("id = ?", leaderboardID).
		Exec(ctx)
	return err
}

// CheckTagAvailability checks if a tag number is currently available in the active leaderboard.
func (db *LeaderboardDBImpl) CheckTagAvailability(ctx context.Context, tagNumber int) (bool, error) {
	var exists bool
	err := db.DB.NewSelect().
		Model((*Leaderboard)(nil)).
		ColumnExpr("EXISTS (SELECT 1 FROM jsonb_each(leaderboard_data) WHERE key = ?)", strconv.Itoa(tagNumber)).
		Where("is_active = ?", true).
		Scan(ctx, &exists)
	if err != nil {
		return false, err
	}

	fmt.Println("CheckTagAvailability: tagNumber =", tagNumber, "exists =", exists) // Add this log

	return !exists, nil
}

// AssignTag assigns a tag to a Discord ID, updates the leaderboard, and sets the source of the update.
func (db *LeaderboardDBImpl) AssignTag(ctx context.Context, discordID leaderboardtypes.DiscordID, tagNumber int, source ServiceUpdateTagSource, updateID string) error {
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get the active leaderboard within the transaction
	leaderboard := new(Leaderboard)
	err = tx.NewSelect().Model(leaderboard).Where("is_active = ?", true).Scan(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active leaderboard: %w", err)
	}

	// Check if the tag is already assigned
	if _, exists := leaderboard.LeaderboardData[tagNumber]; exists {
		return fmt.Errorf("tag %d is already assigned", tagNumber)
	}

	// Update the leaderboard data to include the new assignment
	if leaderboard.LeaderboardData == nil {
		leaderboard.LeaderboardData = make(map[int]string)
	}
	leaderboard.LeaderboardData[tagNumber] = string(discordID)

	// Deactivate the current leaderboard
	_, err = tx.NewUpdate().
		Model((*Leaderboard)(nil)).
		Set("is_active = ?", false).
		Where("id = ?", leaderboard.ID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to deactivate current leaderboard: %w", err)
	}

	// Create a new active leaderboard with the updated data
	newLeaderboard := &Leaderboard{
		LeaderboardData:   leaderboard.LeaderboardData,
		IsActive:          true,
		ScoreUpdateSource: source,
		ScoreUpdateID:     updateID,
	}
	_, err = tx.NewInsert().Model(newLeaderboard).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create new leaderboard: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// UpdateLeaderboard updates the leaderboard with new data from the Score module.
func (db *LeaderboardDBImpl) UpdateLeaderboard(ctx context.Context, leaderboardData map[int]string, scoreUpdateID string) error {
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Safe to call even if tx is committed

	// Deactivate the current active leaderboard
	var activeLeaderboard Leaderboard
	err = tx.NewSelect().Model(&activeLeaderboard).Where("is_active = ?", true).Scan(ctx)
	if err != nil {
		return fmt.Errorf("failed to find active leaderboard: %w", err)
	}

	_, err = tx.NewUpdate().
		Model((*Leaderboard)(nil)).
		Set("is_active = ?", false).
		Where("id = ?", activeLeaderboard.ID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to deactivate current leaderboard: %w", err)
	}

	// Create a new leaderboard with the updated data
	newLeaderboard := &Leaderboard{
		LeaderboardData:   leaderboardData,
		IsActive:          true,
		ScoreUpdateSource: ServiceUpdateTagSourceProcessScores,
		ScoreUpdateID:     scoreUpdateID,
	}

	_, err = tx.NewInsert().Model(newLeaderboard).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to insert new leaderboard: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// SwapTags swaps the tag numbers of two users in the leaderboard.
func (db *LeaderboardDBImpl) SwapTags(ctx context.Context, requestorID, targetID string) error {
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Safe to call even if tx is committed

	// Get the active leaderboard within the transaction
	leaderboard := new(Leaderboard)
	err = tx.NewSelect().Model(leaderboard).Where("is_active = ?", true).Scan(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active leaderboard: %w", err)
	}

	// Find the tag numbers for the requestor and target
	var requestorTag, targetTag int
	var foundRequestor, foundTarget bool

	for tag, discordID := range leaderboard.LeaderboardData {
		if discordID == requestorID {
			requestorTag = tag
			foundRequestor = true
		}
		if discordID == targetID {
			targetTag = tag
			foundTarget = true
		}
	}

	if !foundRequestor || !foundTarget {
		return fmt.Errorf("one or both of the users were not found in the active leaderboard")
	}

	// Swap the Discord IDs in the leaderboard data
	leaderboard.LeaderboardData[requestorTag] = targetID
	leaderboard.LeaderboardData[targetTag] = requestorID

	// Deactivate the current leaderboard
	_, err = tx.NewUpdate().
		Model((*Leaderboard)(nil)).
		Set("is_active = ?", false).
		Where("id = ?", leaderboard.ID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to deactivate current leaderboard: %w", err)
	}

	// Create a new active leaderboard with the updated data
	newLeaderboard := &Leaderboard{
		LeaderboardData:   leaderboard.LeaderboardData,
		IsActive:          true,
		ScoreUpdateSource: ServiceUpdateTagSourceManual, // Assuming manual swap
	}
	_, err = tx.NewInsert().Model(newLeaderboard).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create new leaderboard: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetTagByDiscordID retrieves the tag number associated with a Discord ID from the active leaderboard.
func (db *LeaderboardDBImpl) GetTagByDiscordID(ctx context.Context, discordID string) (int, error) {
	// Get the active leaderboard
	leaderboard := new(Leaderboard)
	err := db.DB.NewSelect().
		Model(leaderboard).
		Where("is_active = ?", true).
		Scan(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get active leaderboard: %w", err)
	}

	// Find the tag number for the given Discord ID
	for tag, id := range leaderboard.LeaderboardData {
		if id == discordID {
			return tag, nil
		}
	}

	return 0, fmt.Errorf("no tag found for discord ID: %s", discordID)
}
