package leaderboardservice

import (
	"context"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// TagSwapRequested matches the Interface: (ctx, guildID, userID, targetTag)
func (s *LeaderboardService) TagSwapRequested(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	userID sharedtypes.DiscordID,
	targetTag sharedtypes.TagNumber,
) (results.OperationResult, error) {
	return s.withTelemetry(ctx, "TagSwapRequested", guildID, func(ctx context.Context) (results.OperationResult, error) {

		return s.runInTx(ctx, func(ctx context.Context, db bun.IDB) (results.OperationResult, error) {
			// 1. Get current leaderboard state inside transaction
			current, err := s.repo.GetActiveLeaderboardIDB(ctx, db, guildID)
			if err != nil {
				return results.FailureResult(&leaderboardevents.TagSwapFailedPayloadV1{GuildID: guildID, Reason: "database error"}), err
			}

			// 2. Find the requestor's current tag
			requestorTag, found := s.FindTagByUserID(current.LeaderboardData, userID)
			if !found {
				return results.FailureResult(&leaderboardevents.TagSwapFailedPayloadV1{GuildID: guildID, Reason: "requesting user not on leaderboard"}), nil
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
				return results.FailureResult(&leaderboardevents.TagSwapFailedPayloadV1{GuildID: guildID, Reason: "target tag not currently assigned"}), nil
			}

			if userID == targetUserID {
				return results.FailureResult(&leaderboardevents.TagSwapFailedPayloadV1{GuildID: guildID, Reason: "cannot swap tag with self"}), nil
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
			_, err = s.repo.UpdateLeaderboard(
				ctx,
				db,
				guildID,
				newData,
				sharedtypes.RoundID(uuid.New()),
				sharedtypes.ServiceUpdateSourceTagSwap,
			)
			if err != nil {
				return results.FailureResult(&leaderboardevents.TagSwapFailedPayloadV1{GuildID: guildID, Reason: "database error"}), err
			}

			// Build success payload (use processed payload defined in shared events)
			payload := &leaderboardevents.TagSwapProcessedPayloadV1{
				GuildID:     guildID,
				RequestorID: userID,
				TargetID:    targetUserID,
			}

			return results.SuccessResult(payload), nil
		})
	})
}
