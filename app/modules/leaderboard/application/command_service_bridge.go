package leaderboardservice

import (
	"context"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboarddomain "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/domain"
)

// ProcessRoundCommand runs the normalized command flow through the service boundary.
func (s *LeaderboardService) ProcessRoundCommand(
	ctx context.Context,
	cmd ProcessRoundCommand,
) (*ProcessRoundOutput, error) {
	if s.commandPipeline == nil {
		return nil, ErrCommandPipelineUnavailable
	}
	return s.commandPipeline.ProcessRound(ctx, cmd)
}

// ResetTagsFromQualifyingRound clears and reassigns tags based on finish order.
func (s *LeaderboardService) ResetTagsFromQualifyingRound(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	finishOrder []sharedtypes.DiscordID,
) ([]leaderboarddomain.TagChange, error) {
	if s.commandPipeline == nil {
		return nil, ErrCommandPipelineUnavailable
	}

	order := make([]string, 0, len(finishOrder))
	for _, memberID := range finishOrder {
		order = append(order, string(memberID))
	}

	return s.commandPipeline.ResetTags(ctx, string(guildID), order)
}

// EndSeason ends the active season for a guild through normalized command orchestration.
func (s *LeaderboardService) EndSeason(ctx context.Context, guildID sharedtypes.GuildID) error {
	if s.commandPipeline == nil {
		return ErrCommandPipelineUnavailable
	}
	return s.commandPipeline.EndSeason(ctx, string(guildID))
}
