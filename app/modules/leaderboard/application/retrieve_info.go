package leaderboardservice

import (
	"context"
	"database/sql"
	"fmt"

	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
)

// TagAvailabilityResult represents the detailed result of a tag availability check.
type TagAvailabilityResult struct {
	Available bool
	Reason    string
}

// GetLeaderboard returns the active leaderboard entries as domain types.
func (s *LeaderboardService) GetLeaderboard(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	seasonID string,
) (results.OperationResult[[]leaderboardtypes.LeaderboardEntry, error], error) {

	return withTelemetry(s, ctx, "GetLeaderboard", guildID, func(ctx context.Context) (results.OperationResult[[]leaderboardtypes.LeaderboardEntry, error], error) {
		if s.commandPipeline == nil {
			return results.OperationResult[[]leaderboardtypes.LeaderboardEntry, error]{}, ErrCommandPipelineUnavailable
		}

		taggedMembers, err := s.commandPipeline.GetTaggedMembers(ctx, string(guildID), nil)
		if err != nil {
			return results.OperationResult[[]leaderboardtypes.LeaderboardEntry, error]{}, err
		}

		entries := make([]leaderboardtypes.LeaderboardEntry, len(taggedMembers))
		for i, member := range taggedMembers {
			entries[i] = leaderboardtypes.LeaderboardEntry{
				UserID:    sharedtypes.DiscordID(member.MemberID),
				TagNumber: sharedtypes.TagNumber(member.Tag),
			}
		}

		// Enrich from seasonal standings where available.
		if err := s.enrichWithSeasonData(ctx, s.db, guildID, seasonID, entries); err != nil {
			// Already logged in helper, continue with unenriched data
		}

		return results.SuccessResult[[]leaderboardtypes.LeaderboardEntry, error](entries), nil
	})
}

// GetTaggedMembers returns the current normalized tag state for a guild, sorted by tag (optionally filtered by club).
func (s *LeaderboardService) GetTaggedMembers(ctx context.Context, guildID sharedtypes.GuildID, clubUUID *string) ([]TaggedMemberView, error) {
	s.logger.DebugContext(ctx, "leaderboard: about to call memberRepo.GetTaggedMembers", "guild_id", guildID)
	// 5. Get current tags (pass s.db)
	repoMembers, err := s.memberRepo.GetTaggedMembers(ctx, s.db, string(guildID), clubUUID)
	s.logger.DebugContext(ctx, "leaderboard: finished memberRepo.GetTaggedMembers", "guild_id", guildID, "member_count", len(repoMembers), "err", err)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get tagged members",
			"guild_id", guildID,
			"club_uuid", clubUUID,
			"error", err.Error(),
		)
		return nil, fmt.Errorf("failed to get tagged members: %w", err)
	}

	// Map DB model to Service View Model
	members := make([]TaggedMemberView, 0, len(repoMembers))
	for _, m := range repoMembers {
		if m.CurrentTag == nil {
			continue // Should be filtered by query but safe to check
		}
		members = append(members, TaggedMemberView{
			MemberID: m.MemberID,
			Tag:      *m.CurrentTag,
		})
	}

	return members, nil
}

// getAllMembersCore returns ALL members for a guild including those without an active tag (optionally filtered by club).
func (s *LeaderboardService) getAllMembersCore(ctx context.Context, guildID sharedtypes.GuildID, clubUUID *string) ([]MemberTagView, error) {
	// 5. Get current tags (pass s.db)
	repoMembers, err := s.memberRepo.GetMembersByGuild(ctx, s.db, string(guildID))
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get all members",
			"guild_id", guildID,
			"club_uuid", clubUUID,
			"error", err.Error(),
		)
		return nil, fmt.Errorf("failed to get all members: %w", err)
	}

	// Map DB model to Service View Model
	members := make([]MemberTagView, 0, len(repoMembers))
	for _, m := range repoMembers {
		// PWA needs to receive Members with nil tags, so no filter here.
		members = append(members, MemberTagView{
			MemberID: m.MemberID,
			Tag:      m.CurrentTag,
		})
	}

	return members, nil
}

// GetTagByUserID returns the tag number for a given user.
func (s *LeaderboardService) GetTagByUserID(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	userID sharedtypes.DiscordID,
) (results.OperationResult[sharedtypes.TagNumber, error], error) {

	return withTelemetry(s, ctx, "GetTagByUserID", guildID, func(ctx context.Context) (results.OperationResult[sharedtypes.TagNumber, error], error) {
		if s.commandPipeline == nil {
			return results.OperationResult[sharedtypes.TagNumber, error]{}, ErrCommandPipelineUnavailable
		}

		tag, found, err := s.commandPipeline.GetMemberTag(ctx, string(guildID), string(userID))
		if err != nil {
			return results.OperationResult[sharedtypes.TagNumber, error]{}, err
		}
		if !found {
			return results.FailureResult[sharedtypes.TagNumber](sql.ErrNoRows), nil
		}
		return results.SuccessResult[sharedtypes.TagNumber, error](sharedtypes.TagNumber(tag)), nil
	})
}

// RoundGetTagByUserID wraps GetTagByUserID for telemetry/results but still returns domain type.
// DEPRECATED: Use GetTagByUserID directly as it now includes telemetry.
// Kept for interface compatibility but updated signature.
func (s *LeaderboardService) RoundGetTagByUserID(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	userID sharedtypes.DiscordID,
) (results.OperationResult[sharedtypes.TagNumber, error], error) {
	return s.GetTagByUserID(ctx, guildID, userID)
}

// CheckTagAvailability returns domain result; handler converts it to payload.
func (s *LeaderboardService) CheckTagAvailability(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	userID sharedtypes.DiscordID,
	tagNumber sharedtypes.TagNumber,
) (results.OperationResult[TagAvailabilityResult, error], error) {

	return withTelemetry(s, ctx, "CheckTagAvailability", guildID, func(ctx context.Context) (results.OperationResult[TagAvailabilityResult, error], error) {
		if s.commandPipeline == nil {
			return results.OperationResult[TagAvailabilityResult, error]{}, ErrCommandPipelineUnavailable
		}

		available, reason, err := s.commandPipeline.CheckTagAvailability(ctx, string(guildID), string(userID), int(tagNumber))
		if err != nil {
			return results.OperationResult[TagAvailabilityResult, error]{}, err
		}
		return results.SuccessResult[TagAvailabilityResult, error](TagAvailabilityResult{Available: available, Reason: reason}), nil
	})
}
