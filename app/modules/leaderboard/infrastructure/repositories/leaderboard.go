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

func (db *LeaderboardDBImpl) GetActiveLeaderboard(ctx context.Context, guildID sharedtypes.GuildID) (*Leaderboard, error) {
	leaderboard := new(Leaderboard)
	err := db.DB.NewSelect().
		Model(leaderboard).
		Column("id", "leaderboard_data", "is_active", "update_source", "update_id", "guild_id").
		Where("is_active = ?", true).
		Where("guild_id = ?", guildID).
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
func (db *LeaderboardDBImpl) CreateLeaderboard(ctx context.Context, guildID sharedtypes.GuildID, leaderboard *Leaderboard) (int64, error) {
	leaderboard.GuildID = guildID
	// Use Bun's Returning to get the inserted ID
	_, err := db.DB.NewInsert().
		Model(leaderboard).
		Returning("id").
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to create leaderboard: %w", err)
	}
	return leaderboard.ID, nil
}

// DeactivateLeaderboard deactivates the specified leaderboard.
func (db *LeaderboardDBImpl) DeactivateLeaderboard(ctx context.Context, guildID sharedtypes.GuildID, leaderboardID int64) error {
	_, err := db.DB.NewUpdate().
		Model((*Leaderboard)(nil)).
		Set("is_active = ?", false).
		Where("id = ?", leaderboardID).
		Where("guild_id = ?", guildID).
		Exec(ctx)
	return err
}

// CheckTagAvailability checks if a tag number is currently available in the active leaderboard.
// It returns a detailed result indicating availability and the specific reason if unavailable:
// 1. The specific tag is already taken by someone else, OR
// 2. The user already has any tag in the leaderboard (no duplicate signups)
func (db *LeaderboardDBImpl) CheckTagAvailability(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, tagNumber sharedtypes.TagNumber) (TagAvailabilityResult, error) {
	leaderboard, err := db.GetActiveLeaderboard(ctx, guildID)
	if err != nil {
		// If no active leaderboard exists for this guild, create an empty one to enable signup flow.
		if errors.Is(err, ErrNoActiveLeaderboard) {
			newLeaderboard := &Leaderboard{
				LeaderboardData: make(leaderboardtypes.LeaderboardData, 0),
				IsActive:        true,
				UpdateSource:    sharedtypes.ServiceUpdateSourceManual,
				GuildID:         guildID,
			}
			if _, createErr := db.DB.NewInsert().Model(newLeaderboard).Exec(ctx); createErr != nil {
				// If creation fails, return the original "no active leaderboard" error
				return TagAvailabilityResult{Available: false}, ErrNoActiveLeaderboard
			}
			leaderboard = newLeaderboard
		} else {
			// Propagate other database errors
			return TagAvailabilityResult{Available: false}, fmt.Errorf("failed to get active leaderboard for tag availability check: %w", err)
		}
	}

	// Check if the user already has any tag (prevent duplicate signups)
	if leaderboard.HasUserID(userID) {
		return TagAvailabilityResult{
			Available: false,
			Reason:    "user already has a tag",
		}, nil
	}

	// Check if the specific tag is available
	if leaderboard.HasTagNumber(tagNumber) {
		return TagAvailabilityResult{
			Available: false,
			Reason:    "tag already taken",
		}, nil
	}

	return TagAvailabilityResult{Available: true}, nil
}

func (l *Leaderboard) HasTagNumber(tagNumber sharedtypes.TagNumber) bool {
	for _, entry := range l.LeaderboardData {
		// Safely check if TagNumber is not nil before dereferencing
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

// AssignTag assigns a tag to a Discord ID, updates the leaderboard, and sets the source of the update.
func (db *LeaderboardDBImpl) AssignTag(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, tagNumber sharedtypes.TagNumber, source string, requestUpdateID sharedtypes.RoundID, requestingUserID sharedtypes.DiscordID) (sharedtypes.RoundID, error) {
	// Validate that tag number is not 0
	if tagNumber == 0 {
		return sharedtypes.RoundID(uuid.Nil), fmt.Errorf("invalid tag assignment: tag number cannot be 0 for user %s", userID)
	}

	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return sharedtypes.RoundID(uuid.Nil), fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	leaderboard := new(Leaderboard)
	err = tx.NewSelect().Model(leaderboard).Where("is_active = ?", true).Where("guild_id = ?", guildID).Scan(ctx)
	if err != nil {
		return sharedtypes.RoundID(uuid.Nil), fmt.Errorf("failed to get active leaderboard during AssignTag: %w", err)
	}

	tagMap := make(map[sharedtypes.TagNumber]sharedtypes.DiscordID)
	for _, entry := range leaderboard.LeaderboardData {
		if entry.TagNumber != 0 {
			tagMap[entry.TagNumber] = entry.UserID
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
			TagNumber: tagValue,
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
		UpdateSource:    sharedtypes.ServiceUpdateSource(source),
		UpdateID:        newAssignmentID,
		GuildID:         guildID,
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

// UpdateLeaderboard updates the leaderboard with new data and returns the updated leaderboard.
func (db *LeaderboardDBImpl) UpdateLeaderboard(ctx context.Context, guildID sharedtypes.GuildID, leaderboardData leaderboardtypes.LeaderboardData, UpdateID sharedtypes.RoundID) (*Leaderboard, error) {
	tx, err := db.DB.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err) // ← RETURN NIL, ERROR
	}
	defer tx.Rollback()

	// 1. Deactivate the current leaderboard
	_, err = tx.NewUpdate().
		Model((*Leaderboard)(nil)).
		Set("is_active = ?", false).
		Where("is_active = ?", true).
		Where("guild_id = ?", guildID).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to deactivate current leaderboard during UpdateLeaderboard: %w", err) // ← RETURN NIL, ERROR
	}

	// 2. Create a new leaderboard with the updated data
	newLeaderboard := &Leaderboard{
		LeaderboardData: leaderboardData,
		IsActive:        true,
		UpdateSource:    sharedtypes.ServiceUpdateSourceProcessScores,
		UpdateID:        UpdateID,
		GuildID:         guildID,
	}

	// Insert and let Bun handle ID population
	_, err = tx.NewInsert().
		Model(newLeaderboard).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to insert new leaderboard during UpdateLeaderboard: %w", err) // ← RETURN NIL, ERROR
	}

	// 3. Commit the transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction during UpdateLeaderboard: %w", err) // ← RETURN NIL, ERROR
	}

	return newLeaderboard, nil // ← RETURN LEADERBOARD, NIL
}

// SwapTags swaps the tag numbers of two users in the leaderboard.
func (db *LeaderboardDBImpl) SwapTags(ctx context.Context, guildID sharedtypes.GuildID, requestorID, targetID sharedtypes.DiscordID) error {
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get the active leaderboard within the transaction
	leaderboard := new(Leaderboard)
	err = tx.NewSelect().Model(leaderboard).Where("is_active = ?", true).Where("guild_id = ?", guildID).Scan(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active leaderboard during SwapTags: %w", err)
	}

	// Convert LeaderboardData to a map for easy tag lookup
	tagMap := make(map[sharedtypes.TagNumber]sharedtypes.DiscordID)
	for _, entry := range leaderboard.LeaderboardData {
		// Safely handle nil TagNumber pointers
		if entry.TagNumber != 0 {
			tagMap[entry.TagNumber] = entry.UserID
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
			TagNumber: tagValue,
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
		UpdateSource:    sharedtypes.ServiceUpdateSourceManual,
		GuildID:         guildID,
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
func (db *LeaderboardDBImpl) GetTagByUserID(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (*sharedtypes.TagNumber, error) {
	// Create a typed query for better compile-time safety
	var activeLeaderboard Leaderboard
	err := db.DB.NewSelect().
		Model(&activeLeaderboard).
		Where("is_active = ?", true).
		Where("guild_id = ?", guildID).
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
		if entry.UserID == userID && entry.TagNumber != 0 {
			tagVal := entry.TagNumber
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
