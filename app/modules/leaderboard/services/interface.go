package leaderboardservices

import (
	"context"

	leaderboardcommands "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/commands"
	leaderboarddto "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/dto"
)

// LeaderboardService handles the logic for managing the leaderboard.
type LeaderboardService struct{}

// NewLeaderboardService creates a new LeaderboardService.
func NewLeaderboardService() *LeaderboardService {
	return &LeaderboardService{}
}

// LeaderboardService defines the interface for managing the leaderboard.
type Service interface {
	AssignTags(ctx context.Context, entries []leaderboarddto.LeaderboardEntry) ([]leaderboarddto.LeaderboardEntry, error)
	UpdateLeaderboardData(currentLeaderboardData map[int]string, newEntries []leaderboarddto.LeaderboardEntry) map[int]string
	InitiateTagSwap(ctx context.Context, swapRequest *leaderboardcommands.TagSwapRequest) (*SwapGroupResult, error)
	RemoveSwapGroup(ctx context.Context, swapGroup *SwapGroup) error
	StoreSwapGroup(ctx context.Context, swapGroup *SwapGroup) error
}
