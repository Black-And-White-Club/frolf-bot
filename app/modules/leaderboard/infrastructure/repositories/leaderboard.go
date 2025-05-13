package leaderboarddb

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

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
	leaderboard := new(Leaderboard)

	err := db.DB.NewSelect().
		Model(leaderboard).
		Column("id", "leaderboard_data", "is_active", "update_source", "update_id").
		Where("is_active = ?", true).
		Limit(1).
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
	leaderboard, err := db.GetActiveLeaderboard(ctx)
	if err != nil {
		// Propagate the error from GetActiveLeaderboard
		return false, fmt.Errorf("failed to get active leaderboard for tag availability check: %w", err)
	}

	return !leaderboard.HasTagNumber(tagNumber), nil
}

func (l *Leaderboard) HasTagNumber(tagNumber sharedtypes.TagNumber) bool {
	for _, entry := range l.LeaderboardData {
		// Safely check if TagNumber is not nil before dereferencing
		if entry.TagNumber != nil && *entry.TagNumber == tagNumber {
			return true
		}
	}
	return false
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
		return fmt.Errorf("failed to get active leaderboard during AssignTag: %w", err)
	}

	// Convert LeaderboardData to a map for easy tag lookup
	tagMap := make(map[sharedtypes.TagNumber]sharedtypes.DiscordID)
	for _, entry := range leaderboard.LeaderboardData {
		// Safely handle nil TagNumber pointers
		if entry.TagNumber != nil {
			tagMap[*entry.TagNumber] = entry.UserID
		}
	}

	// Check if the tag is already assigned
	if _, exists := tagMap[tagNumber]; exists {
		return fmt.Errorf("tag %d is already assigned", tagNumber)
	}

	// Add the new assignment to the map
	tagMap[tagNumber] = userID

	// Convert the map back to LeaderboardData
	updatedLeaderboardData := make(leaderboardtypes.LeaderboardData, 0, len(tagMap))
	for tag, uid := range tagMap {
		// Create a pointer to the tag value
		tagValue := tag
		updatedLeaderboardData = append(updatedLeaderboardData, leaderboardtypes.LeaderboardEntry{
			TagNumber: &tagValue,
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
		return fmt.Errorf("failed to deactivate current leaderboard during AssignTag: %w", err)
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
		return fmt.Errorf("failed to create new leaderboard during AssignTag: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction during AssignTag: %w", err)
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
		return fmt.Errorf("failed to get active leaderboard during BatchAssignTags: %w", err)
	}

	// Convert LeaderboardData to a map for easy tag lookup and updates
	tagMap := make(map[sharedtypes.TagNumber]sharedtypes.DiscordID)
	for _, entry := range leaderboard.LeaderboardData {
		// Safely handle nil TagNumber pointers
		if entry.TagNumber != nil {
			tagMap[*entry.TagNumber] = entry.UserID
		}
	}

	// Process all assignments
	for _, assignment := range assignments {
		tagMap[(assignment.TagNumber)] = assignment.UserID
	}

	// Convert the map back to LeaderboardData
	updatedLeaderboardData := make(leaderboardtypes.LeaderboardData, 0, len(tagMap))
	for tag, uid := range tagMap {
		// Create a pointer to the tag value
		tagValue := tag
		updatedLeaderboardData = append(updatedLeaderboardData, leaderboardtypes.LeaderboardEntry{
			TagNumber: &tagValue,
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
		return fmt.Errorf("failed to deactivate current leaderboard during BatchAssignTags: %w", err)
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
		return fmt.Errorf("failed to create new leaderboard during BatchAssignTags: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction during BatchAssignTags: %w", err)
	}

	return nil
}

// UpdateLeaderboard updates the leaderboard with new data from the Score module.
func (db *LeaderboardDBImpl) UpdateLeaderboard(ctx context.Context, leaderboardData leaderboardtypes.LeaderboardData, UpdateID sharedtypes.RoundID) error {
	tx, err := db.DB.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Deactivate the current leaderboard
	_, err = tx.NewUpdate().
		Model((*Leaderboard)(nil)).
		Set("is_active = ?", false).
		Where("is_active = ?", true).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to deactivate current leaderboard during UpdateLeaderboard: %w", err)
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
		return fmt.Errorf("failed to insert new leaderboard during UpdateLeaderboard: %w", err)
	}

	// 3. Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction during UpdateLeaderboard: %w", err)
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
		return fmt.Errorf("failed to get active leaderboard during SwapTags: %w", err)
	}

	// Convert LeaderboardData to a map for easy tag lookup
	tagMap := make(map[sharedtypes.TagNumber]sharedtypes.DiscordID)
	for _, entry := range leaderboard.LeaderboardData {
		// Safely handle nil TagNumber pointers
		if entry.TagNumber != nil {
			tagMap[*entry.TagNumber] = entry.UserID
		}
	}

	// Find the tag numbers for the requestor and target
	var requestorTag, targetTag sharedtypes.TagNumber
	var foundRequestor, foundTarget bool

	for tag, uid := range tagMap {
		if uid == requestorID {
			requestorTag = tag
			foundRequestor = true
		}
		if uid == targetID {
			targetTag = tag
			foundTarget = true
		}
	}

	if !foundRequestor || !foundTarget {
		return fmt.Errorf("one or both of the users were not found in the active leaderboard")
	}

	// Swap the Discord IDs in the map
	tagMap[requestorTag] = targetID
	tagMap[targetTag] = requestorID

	// Convert the map back to LeaderboardData
	updatedLeaderboardData := make(leaderboardtypes.LeaderboardData, 0, len(tagMap))
	for tag, uid := range tagMap {
		// Create a pointer to the tag value
		tagValue := tag
		updatedLeaderboardData = append(updatedLeaderboardData, leaderboardtypes.LeaderboardEntry{
			TagNumber: &tagValue,
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
		return fmt.Errorf("failed to deactivate current leaderboard during SwapTags: %w", err)
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
		return fmt.Errorf("failed to create new leaderboard during SwapTags: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction during SwapTags: %w", err)
	}

	return nil
}

// GetTagByUserID retrieves the tag number for a given user ID from leaderboard_data.
// Returns sql.ErrNoRows if no tag is found for the user.
func (db *LeaderboardDBImpl) GetTagByUserID(ctx context.Context, userID sharedtypes.DiscordID) (*int, error) {
	var leaderboard Leaderboard
	err := db.DB.NewSelect().
		Model(&leaderboard).
		Where("is_active = ?", true).
		Scan(ctx)
	if err != nil {
		// Wrap any database error other than sql.ErrNoRows
		if err != sql.ErrNoRows {
			slog.Error("❌ Failed to fetch leaderboard entry", slog.Any("error", err))
			return nil, fmt.Errorf("failed to get active leaderboard for tag lookup: %w", err)
		}
		// If sql.ErrNoRows from GetActiveLeaderboard, it means no active leaderboard exists,
		// which is a valid scenario to return sql.ErrNoRows to the service layer.
		slog.Warn("⚠️ No active leaderboard found for tag lookup")
		return nil, sql.ErrNoRows
	}

	slog.Info("📊 Retrieved leaderboard entry",
		slog.Any("leaderboard_data", leaderboard.LeaderboardData))

	for _, entry := range leaderboard.LeaderboardData {
		// Safely check if TagNumber is not nil before dereferencing
		if entry.UserID == userID && entry.TagNumber != nil {
			tagNum := int(*entry.TagNumber)
			slog.Info("✅ Tag found!", slog.Int("tag_number", tagNum))
			return &tagNum, nil
		}
	}

	// If no tag is found for the user in the active leaderboard, return sql.ErrNoRows
	slog.Warn("⚠️ No tag found for user in active leaderboard", slog.String("user_id", string(userID)))
	return nil, sql.ErrNoRows
}
