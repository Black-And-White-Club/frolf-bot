package leaderboarddb

import (
	"context"
)

// LeaderboardDB represents the interface for interacting with the leaderboard database.
type LeaderboardDB interface {
	GetLeaderboard(ctx context.Context) (*Leaderboard, error)
	DeactivateCurrentLeaderboard(ctx context.Context) error
	InsertLeaderboard(ctx context.Context, leaderboardData map[int]string, active bool) error
	UpdateLeaderboard(ctx context.Context, entries map[int]string) error // Takes entries instead of scores
	SwapTags(ctx context.Context, requestorID, targetID string) error
	AssignTag(ctx context.Context, discordID string, tagNumber int) error
	GetTagByDiscordID(ctx context.Context, discordID string) (int, error)
	CheckTagAvailability(ctx context.Context, tagNumber int) (bool, error)

	// This method remains in the interface
	updateLeaderboardData(ctx context.Context, updateFunc func(map[int]string) error) error
}
