package leaderboarddb

import (
	"context"

	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
)

// LeaderboardDB represents the interface for interacting with the leaderboard database.
type LeaderboardDB interface {
	GetActiveLeaderboard(ctx context.Context) (*Leaderboard, error)
	CreateLeaderboard(ctx context.Context, leaderboard *Leaderboard) (int64, error) // Now returns int64
	DeactivateLeaderboard(ctx context.Context, leaderboardID int64) error
	UpdateLeaderboard(ctx context.Context, leaderboardData map[int]string, scoreUpdateID string) error
	SwapTags(ctx context.Context, requestorID, targetID string) error
	AssignTag(ctx context.Context, discordID leaderboardtypes.DiscordID, tagNumber int, source ServiceUpdateTagSource, updateID string) error
	GetTagByDiscordID(ctx context.Context, discordID string) (int, error)
	CheckTagAvailability(ctx context.Context, tagNumber int) (bool, error)
}
