package leaderboarddb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"

	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
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
func (db *LeaderboardDBImpl) AssignTag(ctx context.Context, userID leaderboardtypes.UserID, tagNumber int, source ServiceUpdateTagSource, updateID string) error {
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
	leaderboard.LeaderboardData[tagNumber] = string(userID)

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

	for tag, userID := range leaderboard.LeaderboardData {
		if userID == requestorID {
			requestorTag = tag
			foundRequestor = true
		}
		if userID == targetID {
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

// Returns nil if no tag is found.
// GetTagByUserID retrieves the tag number for a given user ID from leaderboard_data.
func (db *LeaderboardDBImpl) GetTagByUserID(ctx context.Context, userID string) (*int, error) {
	leaderboardEntry := struct {
		LeaderboardData map[string]string `bun:"leaderboard_data,type:jsonb"`
	}{}

	// ✅ Fetch leaderboard entry that contains user ID
	err := db.DB.NewSelect().
		Model(&leaderboardEntry).
		Table("leaderboards").
		Where("leaderboard_data::text LIKE ?", fmt.Sprintf("%%%s%%", userID)). // Ensure user ID is present
		Scan(ctx)
	if err != nil {
		slog.Error("❌ Failed to fetch leaderboard entry", slog.Any("error", err))
		return nil, err
	}

	// ✅ Debug log: Ensure we retrieved the correct JSONB data
	slog.Info("📊 Retrieved leaderboard entry",
		slog.Any("leaderboard_data", leaderboardEntry.LeaderboardData))

	// ✅ Extract tag number
	for tag, uid := range leaderboardEntry.LeaderboardData {
		if uid == userID {
			tagNum, convErr := strconv.Atoi(tag) // Convert tag string to int
			if convErr != nil {
				slog.Error("❌ Failed to convert tag number", slog.String("tag", tag), slog.Any("error", convErr))
				return nil, convErr
			}
			slog.Info("✅ Tag found!", slog.Int("tag_number", tagNum))
			return &tagNum, nil
		}
	}

	// ❌ If no tag is found, log it and return nil
	slog.Warn("⚠️ No tag found for user", slog.String("user_id", userID))
	return nil, nil
}
