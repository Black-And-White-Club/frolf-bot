package leaderboardservice

import (
	"context"
	"errors"
	"fmt"
	"slices"

	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
)

// ExecuteBatchTagAssignment is the public entry point for batch changes.
// It opens its own transaction.
func (s *LeaderboardService) ExecuteBatchTagAssignment(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	requests []sharedtypes.TagAssignmentRequest,
	updateID sharedtypes.RoundID,
	source sharedtypes.ServiceUpdateSource,
) (results.OperationResult[leaderboardtypes.LeaderboardData, error], error) {
	return withTelemetry(s, ctx, "ExecuteBatchTagAssignment", guildID, func(ctx context.Context) (results.OperationResult[leaderboardtypes.LeaderboardData, error], error) {
		if s.commandPipeline == nil {
			return results.OperationResult[leaderboardtypes.LeaderboardData, error]{}, ErrCommandPipelineUnavailable
		}

		data, err := s.commandPipeline.ApplyTagAssignments(ctx, string(guildID), requests, source, updateID)
		if err != nil {
			var swapErr *TagSwapNeededError
			if errors.As(err, &swapErr) {
				return results.FailureResult[leaderboardtypes.LeaderboardData, error](swapErr), nil
			}
			return results.OperationResult[leaderboardtypes.LeaderboardData, error]{}, fmt.Errorf("apply tag assignments: %w", err)
		}

		return results.SuccessResult[leaderboardtypes.LeaderboardData, error](data), nil
	})
}

// GenerateUpdatedSnapshot remains public and pure
func (s *LeaderboardService) GenerateUpdatedSnapshot(
	currentData leaderboardtypes.LeaderboardData,
	requests []sharedtypes.TagAssignmentRequest,
) leaderboardtypes.LeaderboardData {

	tagMap := make(map[sharedtypes.DiscordID]sharedtypes.TagNumber)
	for _, entry := range currentData {
		tagMap[entry.UserID] = entry.TagNumber
	}

	for _, req := range requests {
		tagMap[req.UserID] = req.TagNumber
	}

	newData := make(leaderboardtypes.LeaderboardData, 0, len(tagMap))
	for uid, tag := range tagMap {
		if tag == 0 {
			continue
		}
		newData = append(newData, leaderboardtypes.LeaderboardEntry{
			UserID:    uid,
			TagNumber: tag,
		})
	}

	// Overflow-safe sorting
	slices.SortFunc(newData, func(a, b leaderboardtypes.LeaderboardEntry) int {
		if a.TagNumber < b.TagNumber {
			return -1
		} else if a.TagNumber > b.TagNumber {
			return 1
		}
		return 0
	})

	return newData
}

// FindTagByUserID helper.
func (s *LeaderboardService) FindTagByUserID(leaderboardData leaderboardtypes.LeaderboardData, userID sharedtypes.DiscordID) (sharedtypes.TagNumber, bool) {
	for _, entry := range leaderboardData {
		if entry.UserID == userID {
			return entry.TagNumber, true
		}
	}
	return 0, false
}
