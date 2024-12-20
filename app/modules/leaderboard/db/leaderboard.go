package leaderboarddb

import (
	"context"
	"fmt"
	"strconv"

	leaderboardevents "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/events"
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

	// --- Conversion happens here ---
	// Convert map[int]string to []leaderboardevents.Score
	var scores []leaderboardevents.Score
	for tagNumber, discordID := range leaderboard.LeaderboardData {
		scores = append(scores, leaderboardevents.Score{
			TagNumber: strconv.Itoa(tagNumber),
			DiscordID: discordID,
			Score:     0, // Provide the score if needed
		})
	}

	// 4. Update the leaderboard in the database using the converted 'scores'
	err = lb.UpdateLeaderboard(ctx, scores)
	if err != nil {
		return fmt.Errorf("InsertTagAndDiscordID: failed to update leaderboard: %w", err)
	}

	return nil
}

// UpdateLeaderboard updates the leaderboard with new scores.
func (lb *LeaderboardDBImpl) UpdateLeaderboard(ctx context.Context, scores []leaderboardevents.Score) error {
	// 1. Start a transaction
	tx, err := lb.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// 2. Deactivate the current leaderboard
	if err := lb.DeactivateCurrentLeaderboard(ctx); err != nil {
		return fmt.Errorf("failed to deactivate current leaderboard: %w", err)
	}

	// 3. Create a map for the new leaderboard data
	leaderboardData := make(map[int]string)
	for _, score := range scores {
		tagNumber, err := strconv.Atoi(score.TagNumber)
		if err != nil {
			return fmt.Errorf("failed to convert tag number to int: %w", err)
		}
		leaderboardData[tagNumber] = score.DiscordID
	}

	// 4. Insert the new leaderboard
	if err := lb.InsertLeaderboard(ctx, leaderboardData, true); err != nil {
		return fmt.Errorf("failed to insert new leaderboard: %w", err)
	}

	// 5. Commit the transaction
	return tx.Commit()
}

// AssignTag assigns a tag to a user.
func (lb *LeaderboardDBImpl) AssignTag(ctx context.Context, discordID string, tagNumber int) error {
	// 1. Fetch the leaderboard data
	leaderboard, err := lb.GetLeaderboard(ctx)
	if err != nil {
		return fmt.Errorf("AssignTag: failed to get leaderboard: %w", err)
	}

	// 2. Check if the tag is already taken
	if _, taken := leaderboard.LeaderboardData[tagNumber]; taken {
		return fmt.Errorf("AssignTag: tag number %d is already taken", tagNumber)
	}

	// 3. Add the new tag and Discord ID to the leaderboard data
	leaderboard.LeaderboardData[tagNumber] = discordID

	// 4. Update the leaderboard in the database
	err = lb.updateLeaderboardData(ctx, leaderboard.LeaderboardData) // Use the helper function
	if err != nil {
		return fmt.Errorf("AssignTag: failed to update leaderboard: %w", err)
	}

	return nil
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

	// --- Conversion happens here ---
	// Convert map[int]string to []leaderboardevents.Score
	var scores []leaderboardevents.Score
	for tagNumber, discordID := range leaderboard.LeaderboardData {
		scores = append(scores, leaderboardevents.Score{
			TagNumber: strconv.Itoa(tagNumber),
			DiscordID: discordID,
			Score:     0, // Provide the score if needed
		})
	}

	// 5. Update the leaderboard in the database using the converted 'scores'
	err = lb.UpdateLeaderboard(ctx, scores)
	if err != nil {
		return fmt.Errorf("SwapTags: failed to update leaderboard: %w", err)
	}

	return nil
}

// GetTagByDiscordID retrieves the tag number associated with a Discord ID.
func (lb *LeaderboardDBImpl) GetTagByDiscordID(ctx context.Context, discordID string) (int, error) {
	// 1. Fetch the leaderboard data
	leaderboard, err := lb.GetLeaderboard(ctx)
	if err != nil {
		return 0, fmt.Errorf("GetTagByDiscordID: failed to get leaderboard: %w", err)
	}

	// 2. Find the tag number for the given Discord ID
	for tag, id := range leaderboard.LeaderboardData {
		if id == discordID {
			return tag, nil
		}
	}

	// 3. If the Discord ID is not found, return 0
	return 0, nil
}

// CheckTagAvailability checks if a tag number is available.
func (lb *LeaderboardDBImpl) CheckTagAvailability(ctx context.Context, tagNumber int) (bool, error) {
	// 1. Fetch the leaderboard data
	leaderboard, err := lb.GetLeaderboard(ctx)
	if err != nil {
		return false, fmt.Errorf("CheckTagAvailability: failed to get leaderboard: %w", err)
	}

	// 2. Check if the tag number exists in the leaderboard data
	_, taken := leaderboard.LeaderboardData[tagNumber]

	// 3. Return true if the tag is not taken, false otherwise
	return !taken, nil
}

// Helper function to update leaderboard data within a transaction
func (lb *LeaderboardDBImpl) updateLeaderboardData(ctx context.Context, leaderboardData map[int]string) error {
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
