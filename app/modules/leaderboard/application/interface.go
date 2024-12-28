package leaderboardservice

import (
	"context"

	leaderboardevents "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/events"
)

// LeaderboardService handles leaderboard logic.
type Service interface {
	UpdateLeaderboard(ctx context.Context, event leaderboardevents.LeaderboardUpdateEvent) error
	AssignTag(ctx context.Context, event leaderboardevents.TagAssigned) error
	SwapTags(ctx context.Context, requestorID, targetID string) error
	GetLeaderboard(ctx context.Context) ([]leaderboardevents.LeaderboardEntry, error)
	GetTagByDiscordID(ctx context.Context, discordID string) (int, error)
	CheckTagAvailability(ctx context.Context, tagNumber int) (bool, error)
}
