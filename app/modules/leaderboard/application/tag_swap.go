package leaderboardservice

import (
	"context"
	"errors"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// TagSwapRequested matches the Interface: (ctx, guildID, userID, targetTag)
func (s *LeaderboardService) TagSwapRequested(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	userID sharedtypes.DiscordID,
	targetTag sharedtypes.TagNumber,
) (LeaderboardOperationResult, error) {
	return s.serviceWrapper(ctx, "TagSwapRequested", func(ctx context.Context) (LeaderboardOperationResult, error) {

		return s.runInTx(ctx, func(ctx context.Context, db bun.IDB) (LeaderboardOperationResult, error) {
			// 1. Get current leaderboard state inside transaction
			current, err := s.LeaderboardDB.GetActiveLeaderboardIDB(ctx, db, guildID)
			if err != nil {
				return LeaderboardOperationResult{}, err
			}

			// 2. Find the requestor's current tag
			requestorTag, found := s.FindTagByUserID(current.LeaderboardData, userID)
			if !found {
				return LeaderboardOperationResult{Err: errors.New("requesting user not on leaderboard")}, nil
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
				return LeaderboardOperationResult{Err: errors.New("target tag not currently assigned")}, nil
			}

			if userID == targetUserID {
				return LeaderboardOperationResult{Err: errors.New("cannot swap tag with self")}, nil
			}

			// 4. Construct the batch request for the swap
			requests := []sharedtypes.TagAssignmentRequest{
				{UserID: userID, TagNumber: targetTag},
				{UserID: targetUserID, TagNumber: requestorTag},
			}

			// 5. Instead of calling executeBatchLogic (which would fetch the current leaderboard again),
			//    perform the minimal operations here using the already-fetched `current` to avoid duplicate
			//    repository calls and satisfy test expectations about mock call counts.
			newData := s.GenerateUpdatedSnapshot(current.LeaderboardData, requests)

			// Atomic DB Write
			updatedLB, err := s.LeaderboardDB.UpdateLeaderboard(
				ctx,
				db,
				guildID,
				newData,
				sharedtypes.RoundID(uuid.New()),
				sharedtypes.ServiceUpdateSourceTagSwap,
			)
			if err != nil {
				return LeaderboardOperationResult{}, err
			}

			changes := computeTagChanges(current.LeaderboardData, updatedLB.LeaderboardData, guildID, sharedtypes.ServiceUpdateSourceTagSwap)

			return LeaderboardOperationResult{
				Leaderboard: updatedLB.LeaderboardData,
				TagChanges:  changes,
			}, nil
		})
	})
}
