package leaderboarddb

import (
	"context"

	leaderboardevents "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/events"
)

// LeaderboardDB represents the interface for interacting with the leaderboard database.
type LeaderboardDB interface {
	GetLeaderboard(ctx context.Context) (*Leaderboard, error)
	DeactivateCurrentLeaderboard(ctx context.Context) error
	InsertLeaderboard(ctx context.Context, leaderboardData map[int]string, active bool) error
	UpdateLeaderboard(ctx context.Context, scores []leaderboardevents.Score) error // Updated to accept scores
	SwapTags(ctx context.Context, requestorID, targetID string) error
	AssignTag(ctx context.Context, discordID string, tagNumber int) error
	GetTagByDiscordID(ctx context.Context, discordID string) (int, error)
	CheckTagAvailability(ctx context.Context, tagNumber int) (bool, error)
}
