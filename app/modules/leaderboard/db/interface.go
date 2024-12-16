package leaderboarddb

import (
	"context"
)

// LeaderboardDB represents the interface for interacting with the leaderboard database.
type LeaderboardDB interface {
	GetLeaderboard(ctx context.Context) (*Leaderboard, error)
	DeactivateCurrentLeaderboard(ctx context.Context) error
	InsertLeaderboard(ctx context.Context, leaderboardData map[int]string, active bool) error
	UpdateLeaderboard(ctx context.Context, leaderboardData map[int]string) error
	SwapTags(ctx context.Context, requestorID, targetID string) error
	InsertTagAndDiscordID(ctx context.Context, tagNumber int, discordID string) error // Add this method
}
