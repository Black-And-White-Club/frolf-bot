package leaderboarddb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// LeaderboardRepository handles database operations for leaderboards.
type LeaderboardDBImpl struct {
	DB *bun.DB
}

// GetActiveLeaderboard retrieves the currently active leaderboard.
var (
	ErrNoActiveLeaderboard = errors.New("no active leaderboard found")
	ErrUserTagNotFound     = errors.New("user tag not found in active leaderboard")
)

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
			return nil, ErrNoActiveLeaderboard // Return the custom error
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
func (db *LeaderboardDBImpl) AssignTag(ctx context.Context, userID sharedtypes.DiscordID, tagNumber sharedtypes.TagNumber, source string, requestUpdateID sharedtypes.RoundID, requestingUserID sharedtypes.DiscordID) (sharedtypes.RoundID, error) {
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return sharedtypes.RoundID(uuid.Nil), fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	leaderboard := new(Leaderboard)
	err = tx.NewSelect().Model(leaderboard).Where("is_active = ?", true).Scan(ctx)
	if err != nil {
		return sharedtypes.RoundID(uuid.Nil), fmt.Errorf("failed to get active leaderboard during AssignTag: %w", err)
	}

	tagMap := make(map[sharedtypes.TagNumber]sharedtypes.DiscordID)
	for _, entry := range leaderboard.LeaderboardData {
		if entry.TagNumber != nil {
			tagMap[*entry.TagNumber] = entry.UserID
		}
	}

	if _, exists := tagMap[tagNumber]; exists {
		return sharedtypes.RoundID(uuid.Nil), fmt.Errorf("tag %d is already assigned", tagNumber)
	}

	tagMap[tagNumber] = userID

	updatedLeaderboardData := make(leaderboardtypes.LeaderboardData, 0, len(tagMap))
	for tag, uid := range tagMap {
		tagValue := tag
		updatedLeaderboardData = append(updatedLeaderboardData, leaderboardtypes.LeaderboardEntry{
			TagNumber: &tagValue,
			UserID:    sharedtypes.DiscordID(uid),
		})
	}

	_, err = tx.NewUpdate().
		Model((*Leaderboard)(nil)).
		Set("is_active = ?", false).
		Where("id = ?", leaderboard.ID).
		Exec(ctx)
	if err != nil {
		return sharedtypes.RoundID(uuid.Nil), fmt.Errorf("failed to deactivate current leaderboard during AssignTag: %w", err)
	}

	// Generate a new UUID for this specific assignment action
	newAssignmentID := sharedtypes.RoundID(uuid.New())

	newLeaderboard := &Leaderboard{
		LeaderboardData: updatedLeaderboardData,
		IsActive:        true,
		UpdateSource:    ServiceUpdateSource(source),
		UpdateID:        newAssignmentID,
	}
	_, err = tx.NewInsert().Model(newLeaderboard).Exec(ctx)
	if err != nil {
		return sharedtypes.RoundID(uuid.Nil), fmt.Errorf("failed to create new leaderboard during AssignTag: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return sharedtypes.RoundID(uuid.Nil), fmt.Errorf("failed to commit transaction during AssignTag: %w", err)
	}

	// Return the newly generated assignment ID on success
	return newAssignmentID, nil
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
		LeaderboardData: updatedLeaderboardData,
		IsActive:        true,
		UpdateSource:    source,
		UpdateID:        updateID,
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
		LeaderboardData: updatedLeaderboardData,
		IsActive:        true,
		UpdateSource:    ServiceUpdateSourceManual,
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
func (db *LeaderboardDBImpl) GetTagByUserID(ctx context.Context, userID sharedtypes.DiscordID) (*sharedtypes.TagNumber, error) {
	// Create a typed query for better compile-time safety
	var activeLeaderboard Leaderboard
	err := db.DB.NewSelect().
		Model(&activeLeaderboard).
		Where("is_active = ?", true).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Make sure we're returning the standard error
			slog.WarnContext(ctx, "No active leaderboard found for tag lookup")
			return nil, fmt.Errorf("%w", ErrNoActiveLeaderboard) // Correctly wrap standard error
		}
		return nil, fmt.Errorf("failed to query active leaderboard: %w", err)
	}

	// Query for the user's tag in active leaderboard
	var userTag *sharedtypes.TagNumber
	for _, entry := range activeLeaderboard.LeaderboardData {
		if entry.UserID == userID && entry.TagNumber != nil {
			tagVal := *entry.TagNumber
			userTag = &tagVal
			break
		}
	}

	if userTag == nil {
		slog.WarnContext(ctx, "No tag found for user in active leaderboard data",
			attr.UserID(userID))
		// Return standard error directly, not a new error that wraps it
		return nil, sql.ErrNoRows
	}

	return userTag, nil
}
