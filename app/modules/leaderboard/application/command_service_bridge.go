package leaderboardservice

import (
	"context"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
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
func (s *LeaderboardService) EndSeason(ctx context.Context, guildID sharedtypes.GuildID) (results.OperationResult[bool, error], error) {
	return withTelemetry(s, ctx, "EndSeason", guildID, func(ctx context.Context) (results.OperationResult[bool, error], error) {
		if s.commandPipeline == nil {
			return results.OperationResult[bool, error]{}, ErrCommandPipelineUnavailable
		}
		if err := s.commandPipeline.EndSeason(ctx, string(guildID)); err != nil {
			return results.OperationResult[bool, error]{}, err
		}
		return results.SuccessResult[bool, error](true), nil
	})
}

// GetTagHistory returns tag history for a member or all members.
func (s *LeaderboardService) GetTagHistory(ctx context.Context, guildID sharedtypes.GuildID, memberID string, limit int) ([]TagHistoryView, error) {
	if s.commandPipeline == nil {
		return nil, ErrCommandPipelineUnavailable
	}
	return s.commandPipeline.GetTagHistory(ctx, string(guildID), memberID, limit)
}

// GetTagList returns the master tag list for a guild.
// Delegates to GetTaggedMembers — same underlying data, distinct semantic name for the PWA.
func (s *LeaderboardService) GetTagList(ctx context.Context, guildID sharedtypes.GuildID) ([]TaggedMemberView, error) {
	if s.commandPipeline == nil {
		return nil, ErrCommandPipelineUnavailable
	}
	return s.commandPipeline.GetTaggedMembers(ctx, string(guildID))
}

// GenerateTagGraphPNG generates a PNG chart of a member's tag history.
func (s *LeaderboardService) GenerateTagGraphPNG(ctx context.Context, guildID sharedtypes.GuildID, memberID string) ([]byte, error) {
	if s.commandPipeline == nil {
		return nil, ErrCommandPipelineUnavailable
	}
	return s.commandPipeline.GenerateTagGraphPNG(ctx, string(guildID), memberID)
}
