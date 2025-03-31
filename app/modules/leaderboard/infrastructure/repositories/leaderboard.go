package leaderboarddb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"

	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/uptrace/bun"
)

// LeaderboardRepository handles database operations for leaderboards.
type LeaderboardDBImpl struct {
	DB *bun.DB
}

// GetActiveLeaderboard retrieves the currently active leaderboard.
func (db *LeaderboardDBImpl) GetActiveLeaderboard(ctx context.Context) (*Leaderboard, error) {
	// Pre-allocate leaderboard to avoid unnecessary allocations
	leaderboard := new(Leaderboard)

	// Select only needed columns instead of all columns
	err := db.DB.NewSelect().
		Model(leaderboard).
		Column("id", "leaderboard_data", "is_active", "score_update_source", "score_update_id").
		Where("is_active = ?", true).
		Limit(1). // Add limit since we know there should be only one active leaderboard
		Scan(ctx)

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
func (db *LeaderboardDBImpl) CheckTagAvailability(ctx context.Context, tagNumber sharedtypes.TagNumber) (bool, error) {
	var exists bool
	err := db.DB.NewSelect().
		Model((*Leaderboard)(nil)).
		ColumnExpr("EXISTS (SELECT 1 FROM jsonb_each(leaderboard_data) WHERE key = ?)", strconv.Itoa(int(tagNumber))).
		Where("is_active = ?", true).
		Scan(ctx, &exists)
	if err != nil {
		return false, err
	}

	return !exists, nil
}

// AssignTag assigns a tag to a Discord ID, updates the leaderboard, and sets the source of the update.
func (db *LeaderboardDBImpl) AssignTag(ctx context.Context, userID sharedtypes.DiscordID, tagNumber sharedtypes.TagNumber, source ServiceUpdateSource, updateID sharedtypes.RoundID) error {
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

	// Convert LeaderboardData to a map for easy tag lookup
	tagMap := make(map[int]string)
	for _, entry := range leaderboard.LeaderboardData {
		tagMap[int(entry.TagNumber)] = string(entry.UserID)
	}

	// Check if the tag is already assigned
	if _, exists := tagMap[int(tagNumber)]; exists {
		return fmt.Errorf("tag %d is already assigned", tagNumber)
	}

	// Add the new assignment to the map
	tagMap[int(tagNumber)] = string(userID)

	// Convert the map back to LeaderboardData
	updatedLeaderboardData := make(leaderboardtypes.LeaderboardData, 0, len(tagMap))
	for tag, uid := range tagMap {
		updatedLeaderboardData = append(updatedLeaderboardData, leaderboardtypes.LeaderboardEntry{
			TagNumber: sharedtypes.TagNumber(tag),
			UserID:    sharedtypes.DiscordID(uid),
		})
	}

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
		LeaderboardData:     updatedLeaderboardData,
		IsActive:            true,
		UpdateSource:        source,
		UpdateID:            updateID,
		RequestingDiscordID: userID,
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

// BatchAssignTags assigns multiple tags in a single transaction
func (db *LeaderboardDBImpl) BatchAssignTags(ctx context.Context, assignments []TagAssignment, source ServiceUpdateSource, updateID sharedtypes.RoundID, userID sharedtypes.DiscordID) error {
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

	// Convert LeaderboardData to a map for easy tag lookup and updates
	tagMap := make(map[int]string)
	for _, entry := range leaderboard.LeaderboardData {
		tagMap[int(entry.TagNumber)] = string(entry.UserID)
	}

	// Process all assignments
	for _, assignment := range assignments {
		tagMap[int(assignment.TagNumber)] = string(assignment.UserID)
	}

	// Convert the map back to LeaderboardData
	updatedLeaderboardData := make(leaderboardtypes.LeaderboardData, 0, len(tagMap))
	for tag, uid := range tagMap {
		updatedLeaderboardData = append(updatedLeaderboardData, leaderboardtypes.LeaderboardEntry{
			TagNumber: sharedtypes.TagNumber(tag),
			UserID:    sharedtypes.DiscordID(uid),
		})
	}

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
		LeaderboardData:     updatedLeaderboardData,
		IsActive:            true,
		UpdateSource:        source,
		UpdateID:            updateID,
		RequestingDiscordID: userID,
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
func (db *LeaderboardDBImpl) UpdateLeaderboard(ctx context.Context, leaderboardData leaderboardtypes.LeaderboardData, UpdateID sharedtypes.RoundID) error {
	// Start a new transaction
	tx, err := db.DB.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted, // Specify appropriate isolation level
	})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Ensure rollback on function exit if not committed

	// 1. Find the active leaderboard ID with a more efficient query
	var activeLeaderboardID int64
	err = tx.NewSelect().
		Model((*Leaderboard)(nil)).
		Column("id").
		Where("is_active = ?", true).
		Limit(1).
		Scan(ctx, &activeLeaderboardID)

	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to find active leaderboard: %w", err)
	}

	// If there is an active leaderboard, deactivate it
	if err != sql.ErrNoRows {
		_, err = tx.NewUpdate().
			Model((*Leaderboard)(nil)).
			Set("is_active = ?", false).
			Where("id = ?", activeLeaderboardID).
			Exec(ctx)

		if err != nil {
			return fmt.Errorf("failed to deactivate current leaderboard: %w", err)
		}
	}

	// 2. Create a new leaderboard with the updated data
	newLeaderboard := &Leaderboard{
		LeaderboardData: leaderboardData,
		IsActive:        true,
		UpdateSource:    ServiceUpdateSourceProcessScores,
		UpdateID:        UpdateID,
	}

	// Insert the new leaderboard
	_, err = tx.NewInsert().Model(newLeaderboard).Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to insert new leaderboard: %w", err)
	}

	// 3. Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// SwapTags swaps the tag numbers of two users in the leaderboard.
func (db *LeaderboardDBImpl) SwapTags(ctx context.Context, requestorID, targetID sharedtypes.DiscordID) error {
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

	// Convert LeaderboardData to a map for easy tag lookup
	tagMap := make(map[int]string)
	for _, entry := range leaderboard.LeaderboardData {
		tagMap[int(entry.TagNumber)] = string(entry.UserID)
	}

	// Find the tag numbers for the requestor and target
	var requestorTag, targetTag int
	var foundRequestor, foundTarget bool

	for tag, uid := range tagMap {
		if uid == string(requestorID) {
			requestorTag = tag
			foundRequestor = true
		}
		if uid == string(targetID) {
			targetTag = tag
			foundTarget = true
		}
	}

	if !foundRequestor || !foundTarget {
		return fmt.Errorf("one or both of the users were not found in the active leaderboard")
	}

	// Swap the Discord IDs in the map
	tagMap[requestorTag] = string(targetID)
	tagMap[targetTag] = string(requestorID)

	// Convert the map back to LeaderboardData
	updatedLeaderboardData := make(leaderboardtypes.LeaderboardData, 0, len(tagMap))
	for tag, uid := range tagMap {
		updatedLeaderboardData = append(updatedLeaderboardData, leaderboardtypes.LeaderboardEntry{
			TagNumber: sharedtypes.TagNumber(tag),
			UserID:    sharedtypes.DiscordID(uid),
		})
	}

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
		LeaderboardData:     updatedLeaderboardData,
		IsActive:            true,
		UpdateSource:        ServiceUpdateSourceManual,
		RequestingDiscordID: requestorID,
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
func (db *LeaderboardDBImpl) GetTagByUserID(ctx context.Context, userID sharedtypes.DiscordID) (*int, error) {
	leaderboardEntry := struct {
		LeaderboardData map[string]string `bun:"leaderboard_data,type:jsonb"`
	}{}

	err := db.DB.NewSelect().
		Model(&leaderboardEntry).
		Table("leaderboards").
		Where("leaderboard_data::text LIKE ?", fmt.Sprintf("%%%s%%", string(userID))).
		Scan(ctx)
	if err != nil {
		slog.Error("❌ Failed to fetch leaderboard entry", slog.Any("error", err))
		return nil, err
	}

	slog.Info("📊 Retrieved leaderboard entry",
		slog.Any("leaderboard_data", leaderboardEntry.LeaderboardData))

	for tag, uid := range leaderboardEntry.LeaderboardData {
		if uid == string(userID) {
			tagNum, convErr := strconv.Atoi(tag)
			if convErr != nil {
				slog.Error("❌ Failed to convert tag number", slog.String("tag", tag), slog.Any("error", convErr))
				return nil, convErr
			}
			slog.Info("✅ Tag found!", slog.Int("tag_number", tagNum))
			return &tagNum, nil
		}
	}

	// ❌ If no tag is found, log it and return nil
	slog.Warn("⚠️ No tag found for user", slog.String("user_id", string(userID)))
	return nil, nil
}
