package leaderboardservice

import (
	"context"
	"errors"
	"fmt"

	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/google/uuid"
)

// TagSwapRequested performs a manual swap between two users and returns the updated leaderboard data.
func (s *LeaderboardService) TagSwapRequested(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	userID sharedtypes.DiscordID,
	targetTag sharedtypes.TagNumber,
) (results.OperationResult[leaderboardtypes.LeaderboardData, error], error) {
	return withTelemetry(s, ctx, "TagSwapRequested", guildID, func(ctx context.Context) (results.OperationResult[leaderboardtypes.LeaderboardData, error], error) {
		if s.commandPipeline == nil {
			return results.OperationResult[leaderboardtypes.LeaderboardData, error]{}, ErrCommandPipelineUnavailable
		}

		requestorTagResult, err := s.GetTagByUserID(ctx, guildID, userID)
		if err != nil {
			return results.OperationResult[leaderboardtypes.LeaderboardData, error]{}, err
		}
		if requestorTagResult.IsFailure() {
			return results.FailureResult[leaderboardtypes.LeaderboardData, error](fmt.Errorf("requesting user not on leaderboard")), nil
		}
		requestorTag := *requestorTagResult.Success

		leaderboardResult, err := s.GetLeaderboard(ctx, guildID)
		if err != nil {
			return results.OperationResult[leaderboardtypes.LeaderboardData, error]{}, err
		}
		if leaderboardResult.IsFailure() {
			return results.FailureResult[leaderboardtypes.LeaderboardData, error](fmt.Errorf("leaderboard unavailable")), nil
		}

		var targetUserID sharedtypes.DiscordID
		targetFound := false
		for _, entry := range *leaderboardResult.Success {
			if entry.TagNumber == targetTag {
				targetUserID = entry.UserID
				targetFound = true
				break
			}
		}
		if !targetFound {
			return results.FailureResult[leaderboardtypes.LeaderboardData, error](fmt.Errorf("target tag not currently assigned")), nil
		}
		if targetUserID == userID {
			return results.FailureResult[leaderboardtypes.LeaderboardData, error](fmt.Errorf("cannot swap tag with self")), nil
		}

		assignments := []sharedtypes.TagAssignmentRequest{
			{UserID: userID, TagNumber: targetTag},
			{UserID: targetUserID, TagNumber: requestorTag},
		}
		data, err := s.commandPipeline.ApplyTagAssignments(ctx, string(guildID), assignments, sharedtypes.ServiceUpdateSourceTagSwap, sharedtypes.RoundID(uuid.New()))
		if err != nil {
			var swapErr *TagSwapNeededError
			if errors.As(err, &swapErr) {
				return results.FailureResult[leaderboardtypes.LeaderboardData, error](swapErr), nil
			}
			return results.OperationResult[leaderboardtypes.LeaderboardData, error]{}, err
		}

		return results.SuccessResult[leaderboardtypes.LeaderboardData, error](data), nil
	})
}
