package bundb

import (
	"context"
	"fmt"

	"github.com/Black-And-White-Club/tcr-bot/app/models"
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

// GetUserTag retrieves the tag information for a user.
func (lb *leaderboardDB) GetUserTag(ctx context.Context, discordID string) (*models.Leaderboard, error) {
	leaderboard, err := lb.GetLeaderboard(ctx)
	if err != nil {
		return nil, err
	}

	for _, entry := range leaderboard.LeaderboardData {
		if entry.DiscordID == discordID {
			return &models.Leaderboard{
				LeaderboardData: leaderboard.LeaderboardData,
			}, nil
		}
	}

	return nil, nil
}

// IsTagAvailable checks if a tag number is available.
func (lb *leaderboardDB) IsTagAvailable(ctx context.Context, tagNumber int) (bool, error) {
	count, err := lb.db.NewSelect().
		Model((*models.LeaderboardEntry)(nil)).
		Where("tag_number = ?", tagNumber).
		Count(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to check tag availability: %w", err)
	}
	return count == 0, nil
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
func (lb *leaderboardDB) InsertLeaderboard(ctx context.Context, leaderboard *models.Leaderboard) error {
	_, err := lb.db.NewInsert().
		Model(leaderboard).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to insert new leaderboard: %w", err)
	}
	return nil
}
