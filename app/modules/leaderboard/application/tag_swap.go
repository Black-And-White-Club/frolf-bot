package leaderboardservice

import (
	"context"
	"database/sql"
	"fmt"

	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// TagSwapRequested performs a manual swap between two users and returns the updated leaderboard data.
func (s *LeaderboardService) TagSwapRequested(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	userID sharedtypes.DiscordID,
	targetTag sharedtypes.TagNumber,
) (leaderboardtypes.LeaderboardData, error) {
	// Run within a DB transaction if we have a DB configured, otherwise execute directly.
	if s.db == nil {
		return s.tagSwapLogic(ctx, nil, guildID, userID, targetTag)
	}

	var result leaderboardtypes.LeaderboardData
	err := s.db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		var innerErr error
		result, innerErr = s.tagSwapLogic(ctx, tx, guildID, userID, targetTag)
		return innerErr
	})

	return result, err
}

// tagSwapLogic contains the core logic and is DB-aware (accepts bun.IDB).
func (s *LeaderboardService) tagSwapLogic(
	ctx context.Context,
	db bun.IDB,
	guildID sharedtypes.GuildID,
	userID sharedtypes.DiscordID,
	targetTag sharedtypes.TagNumber,
) (leaderboardtypes.LeaderboardData, error) {
	// 1. Get current leaderboard state inside transaction
	current, err := s.repo.GetActiveLeaderboard(ctx, db, guildID)
	if err != nil {
		return nil, err
	}

	// 2. Find the requestor's current tag
	requestorTag, found := s.FindTagByUserID(current.LeaderboardData, userID)
	if !found {
		return nil, fmt.Errorf("requesting user not on leaderboard")
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
		return nil, fmt.Errorf("target tag not currently assigned")
	}

	if userID == targetUserID {
		return nil, fmt.Errorf("cannot swap tag with self")
	}

	// 4. Construct the batch request for the swap
	requests := []sharedtypes.TagAssignmentRequest{
		{UserID: userID, TagNumber: targetTag},
		{UserID: targetUserID, TagNumber: requestorTag},
	}

	// 5. Generate snapshot and write
	newData := s.GenerateUpdatedSnapshot(current.LeaderboardData, requests)

	updatedLB, err := s.repo.UpdateLeaderboard(ctx, db, guildID, newData, sharedtypes.RoundID(uuid.New()), sharedtypes.ServiceUpdateSourceTagSwap)
	if err != nil {
		return nil, err
	}

	return updatedLB.LeaderboardData, nil
}
