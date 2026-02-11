package leaderboardservice

import (
	"context"

	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboarddomain "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/domain"
)

// serviceCommandPipeline delegates command-flow operations to LeaderboardService methods.
// This keeps a single service implementation while preserving a test seam.
type serviceCommandPipeline struct {
	service *LeaderboardService
}

func (p *serviceCommandPipeline) ProcessRound(ctx context.Context, cmd ProcessRoundCommand) (*ProcessRoundOutput, error) {
	return p.service.processRoundCommandCore(ctx, cmd)
}

func (p *serviceCommandPipeline) ApplyTagAssignments(
	ctx context.Context,
	guildID string,
	requests []sharedtypes.TagAssignmentRequest,
	source sharedtypes.ServiceUpdateSource,
	updateID sharedtypes.RoundID,
) (leaderboardtypes.LeaderboardData, error) {
	return p.service.applyTagAssignmentsCore(ctx, guildID, requests, source, updateID)
}

func (p *serviceCommandPipeline) StartSeason(ctx context.Context, guildID, seasonID, seasonName string) error {
	return p.service.startSeasonCore(ctx, guildID, seasonID, seasonName)
}

func (p *serviceCommandPipeline) EndSeason(ctx context.Context, guildID string) error {
	return p.service.endSeasonCore(ctx, guildID)
}

func (p *serviceCommandPipeline) ResetTags(ctx context.Context, guildID string, finishOrder []string) ([]leaderboarddomain.TagChange, error) {
	return p.service.resetTagsCore(ctx, guildID, finishOrder)
}

func (p *serviceCommandPipeline) GetTaggedMembers(ctx context.Context, guildID string) ([]TaggedMemberView, error) {
	return p.service.getTaggedMembersCore(ctx, guildID)
}

func (p *serviceCommandPipeline) GetMemberTag(ctx context.Context, guildID, memberID string) (int, bool, error) {
	return p.service.getMemberTagCore(ctx, guildID, memberID)
}

func (p *serviceCommandPipeline) CheckTagAvailability(ctx context.Context, guildID, memberID string, tagNumber int) (bool, string, error) {
	return p.service.checkTagAvailabilityCore(ctx, guildID, memberID, tagNumber)
}

var _ CommandPipeline = (*serviceCommandPipeline)(nil)
