package leaderboardservice

import (
	"context"
	"fmt"

	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// TagSwapRequested performs a manual swap between two users and returns the updated leaderboard data.
func (s *LeaderboardService) TagSwapRequested(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	userID sharedtypes.DiscordID,
	targetTag sharedtypes.TagNumber,
) (results.OperationResult[leaderboardtypes.LeaderboardData, error], error) {

	// Named transaction function for observability
	tagSwapTx := func(ctx context.Context, db bun.IDB) (results.OperationResult[leaderboardtypes.LeaderboardData, error], error) {
		return s.tagSwapLogic(ctx, db, guildID, userID, targetTag)
	}

	// Wrap with telemetry & transaction
	return withTelemetry(s, ctx, "TagSwapRequested", guildID, func(ctx context.Context) (results.OperationResult[leaderboardtypes.LeaderboardData, error], error) {
		return runInTx(s, ctx, tagSwapTx)
	})
}

// tagSwapLogic contains the core logic and is DB-aware (accepts bun.IDB).
func (s *LeaderboardService) tagSwapLogic(
	ctx context.Context,
	db bun.IDB,
	guildID sharedtypes.GuildID,
	userID sharedtypes.DiscordID,
	targetTag sharedtypes.TagNumber,
) (results.OperationResult[leaderboardtypes.LeaderboardData, error], error) {
	// 1. Get current leaderboard state inside transaction
	current, err := s.repo.GetActiveLeaderboard(ctx, db, guildID)
	if err != nil {
		return results.OperationResult[leaderboardtypes.LeaderboardData, error]{}, err
	}

	// 2. Find the requestor's current tag
	requestorTag, found := s.FindTagByUserID(current.LeaderboardData, userID)
	if !found {
		return results.FailureResult[leaderboardtypes.LeaderboardData, error](fmt.Errorf("requesting user not on leaderboard")), nil
	}

	// 3. Identify who currently holds the target tag
	var targetUserID sharedtypes.DiscordID
	targetFound := false
	for _, entry := range current.LeaderboardData {
		if entry.TagNumber == targetTag {
			targetUserID = entry.UserID
			targetFound = true
			break
		}
	}

	if !targetFound {
		return results.FailureResult[leaderboardtypes.LeaderboardData, error](fmt.Errorf("target tag not currently assigned")), nil
	}

	if userID == targetUserID {
		return results.FailureResult[leaderboardtypes.LeaderboardData, error](fmt.Errorf("cannot swap tag with self")), nil
	}

	// 4. Construct the batch request for the swap
	requests := []sharedtypes.TagAssignmentRequest{
		{UserID: userID, TagNumber: targetTag},
		{UserID: targetUserID, TagNumber: requestorTag},
	}

	// 5. Generate snapshot and write
	newData := s.GenerateUpdatedSnapshot(current.LeaderboardData, requests)

	updatedLB := &leaderboardtypes.Leaderboard{
		GuildID:         guildID,
		LeaderboardData: newData,
		UpdateSource:    sharedtypes.ServiceUpdateSourceTagSwap,
		UpdateID:        sharedtypes.RoundID(uuid.New()),
	}

	if err := s.repo.SaveLeaderboard(ctx, db, updatedLB); err != nil {
		return results.OperationResult[leaderboardtypes.LeaderboardData, error]{}, err
	}

	return results.SuccessResult[leaderboardtypes.LeaderboardData, error](updatedLB.LeaderboardData), nil
}
