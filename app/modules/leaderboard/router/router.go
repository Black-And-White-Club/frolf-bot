package leaderboardrouter

import (
	"context"

	leaderboardcommands "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/commands"
	leaderboarddto "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/dto"
	"github.com/Black-And-White-Club/tcr-bot/internal/commands"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
)

// LeaderboardCommandBus is the command bus for the leaderboard module.
type LeaderboardCommandBus struct {
	publisher message.Publisher
	marshaler cqrs.CommandEventMarshaler
}

// NewLeaderboardCommandBus creates a new LeaderboardCommandBus.
func NewLeaderboardCommandBus(publisher message.Publisher, marshaler cqrs.CommandEventMarshaler) *LeaderboardCommandBus {
	return &LeaderboardCommandBus{publisher: publisher, marshaler: marshaler}
}

func (l LeaderboardCommandBus) Send(ctx context.Context, cmd commands.Command) error {
	return watermillutil.SendCommand(ctx, l.publisher, l.marshaler, cmd, cmd.CommandName())
}

// LeaderboardCommandRouter implements the CommandRouter interface.
type LeaderboardCommandRouter struct {
	commandBus CommandBus
}

// NewLeaderboardCommandRouter creates a new LeaderboardCommandRouter.
func NewLeaderboardCommandRouter(commandBus CommandBus) CommandRouter {
	return &LeaderboardCommandRouter{commandBus: commandBus}
}

// GetLeaderboard implements the CommandService interface.
func (l *LeaderboardCommandRouter) GetLeaderboard(ctx context.Context) error {
	getLeaderboardCmd := leaderboardcommands.GetLeaderboardRequest{}
	return l.commandBus.Send(ctx, getLeaderboardCmd)
}

// UpdateLeaderboard implements the CommandService interface.
func (l *LeaderboardCommandRouter) UpdateLeaderboard(ctx context.Context, input leaderboarddto.UpdateLeaderboardInput) error {
	updateLeaderboardCmd := leaderboardcommands.UpdateLeaderboardRequest{
		Input: input,
	}
	return l.commandBus.Send(ctx, updateLeaderboardCmd)
}

// ReceiveScores implements the CommandService interface.
func (l *LeaderboardCommandRouter) ReceiveScores(ctx context.Context, input leaderboarddto.ReceiveScoresInput) error {
	receiveScoresCmd := leaderboardcommands.ReceiveScoresRequest{
		Input: input,
	}
	return l.commandBus.Send(ctx, receiveScoresCmd)
}

// AssignTags implements the CommandService interface.
func (l *LeaderboardCommandRouter) AssignTags(ctx context.Context, input leaderboarddto.AssignTagsInput) error {
	assignTagsCmd := leaderboardcommands.AssignTagsRequest{
		Input: input,
	}
	return l.commandBus.Send(ctx, assignTagsCmd)
}

// InitiateTagSwap implements the CommandService interface.
func (l *LeaderboardCommandRouter) InitiateTagSwap(ctx context.Context, input leaderboarddto.InitiateTagSwapInput) error {
	initiateTagSwapCmd := leaderboardcommands.InitiateTagSwapRequest{
		Input: input,
	}
	return l.commandBus.Send(ctx, initiateTagSwapCmd)
}

// SwapGroups implements the CommandService interface.
func (l *LeaderboardCommandRouter) SwapGroups(ctx context.Context, input leaderboarddto.SwapGroupsInput) error {
	swapGroupsCmd := leaderboardcommands.SwapGroupsRequest{
		Input: input,
	}
	return l.commandBus.Send(ctx, swapGroupsCmd)
}
