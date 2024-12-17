package leaderboardrouter

import (
	"context"

	leaderboarddto "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/dto"
	"github.com/Black-And-White-Club/tcr-bot/internal/commands"
)

// LeaderboardRouter defines the interface for the leaderboard router.
type CommandRouter interface {
	GetLeaderboard(ctx context.Context) error
	UpdateLeaderboard(ctx context.Context, input leaderboarddto.UpdateLeaderboardInput) error
	ReceiveScores(ctx context.Context, input leaderboarddto.ReceiveScoresInput) error
	AssignTags(ctx context.Context, input leaderboarddto.AssignTagsInput) error
	InitiateTagSwap(ctx context.Context, input leaderboarddto.InitiateTagSwapInput) error
	SwapGroups(ctx context.Context, input leaderboarddto.SwapGroupsInput) error
}

// CommandBus is the interface for the command bus.
type CommandBus interface {
	Send(ctx context.Context, cmd commands.Command) error
}
