package leaderboardservice

import (
	"context"
	"errors"
	"slices"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/uptrace/bun"
)

// ExecuteBatchTagAssignment is the public entry point for batch changes.
// It opens its own transaction.
func (s *LeaderboardService) ExecuteBatchTagAssignment(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	requests []sharedtypes.TagAssignmentRequest,
	updateID sharedtypes.RoundID,
	source sharedtypes.ServiceUpdateSource,
) (results.OperationResult, error) {

	return s.withTelemetry(ctx, "ExecuteBatchTagAssignment", guildID, func(ctx context.Context) (results.OperationResult, error) {
		// 1. Transactional Execution
		return s.runInTx(ctx, func(ctx context.Context, db bun.IDB) (results.OperationResult, error) {
			// 2. Call the internal logic helper
			return s.executeBatchLogic(ctx, db, guildID, requests, updateID, source)
		})
	})
}

// executeBatchLogic contains the core "Funnel" logic.
// It takes a bun.IDB so it can run inside or outside an existing transaction.
func (s *LeaderboardService) executeBatchLogic(
	ctx context.Context,
	db bun.IDB,
	guildID sharedtypes.GuildID,
	requests []sharedtypes.TagAssignmentRequest,
	updateID sharedtypes.RoundID,
	source sharedtypes.ServiceUpdateSource,
)(results.OperationResult, error) {

	s.logInfoContext(ctx, "Executing funnel logic",
		attr.String("source", string(source)),
		attr.String("update_id", updateID.String()),
		attr.Int("request_count", len(requests)),
	)

	// 1. Get current state (IDB aware)
	current, err := s.repo.GetActiveLeaderboardIDB(ctx, db, guildID)
	if err != nil {
		if errors.Is(err, leaderboarddb.ErrNoActiveLeaderboard) {
			current = &leaderboarddb.Leaderboard{
				LeaderboardData: leaderboardtypes.LeaderboardData{},
			}
		} else {
			return results.FailureResult(&leaderboardevents.LeaderboardBatchTagAssignmentFailedPayloadV1{GuildID: guildID, Reason: "database error"}), err
		}
	}
	beforeData := current.LeaderboardData

	// --- 2. CONFLICT DETECTION ---
	// Map existing tags to users for ownership lookup
	tagToUserMap := make(map[sharedtypes.TagNumber]sharedtypes.DiscordID)
	// Map users to their current tags for error metadata
	userToTagMap := make(map[sharedtypes.DiscordID]sharedtypes.TagNumber)

	for _, entry := range beforeData {
		tagToUserMap[entry.TagNumber] = entry.UserID
		userToTagMap[entry.UserID] = entry.TagNumber
	}

	// Identify who is in the current batch to allow internal swaps
	requestingUsers := make(map[sharedtypes.DiscordID]bool)
	for _, req := range requests {
		requestingUsers[req.UserID] = true
	}

	for _, req := range requests {
		// Does someone currently hold the tag we want?
		if holderID, occupied := tagToUserMap[req.TagNumber]; occupied {
			// CONFLICT: Tag held by someone NOT in this update batch
			if !requestingUsers[holderID] {
				s.logInfoContext(ctx, "Tag conflict detected, redirection to Saga required",
					attr.Int("tag", int(req.TagNumber)),
					attr.String("requestor", string(req.UserID)),
					attr.String("current_holder", string(holderID)),
				)

				// Find the requestor's current tag to help the Saga map the chain
				currentTag := userToTagMap[req.UserID]

				return results.OperationResult{}, &TagSwapNeededError{
					RequestorID:  req.UserID,
					TargetUserID: holderID,
					TargetTag:    req.TagNumber,
					CurrentTag:   currentTag,
				}
			}
		}
	}

	// --- 3. EXECUTION ---
	// Generate updated snapshot (pure logic)
	newData := s.GenerateUpdatedSnapshot(beforeData, requests)

	// Atomic DB Write
	updatedLB, err := s.repo.UpdateLeaderboard(
		ctx,
		db,
		guildID,
		newData,
		updateID,
		source,
	)
	if err != nil {
		return results.FailureResult(&leaderboardevents.LeaderboardBatchTagAssignmentFailedPayloadV1{GuildID: guildID, Reason: "database error"}), err
	}

	// 4. Build success payload with leaderboard data
	payload := &leaderboardevents.LeaderboardBatchTagAssignedPayloadV1{
		GuildID:         guildID,
		AssignmentCount: len(updatedLB.LeaderboardData),
		Assignments:     make([]leaderboardevents.TagAssignmentInfoV1, len(updatedLB.LeaderboardData)),
	}

	for i, entry := range updatedLB.LeaderboardData {
		payload.Assignments[i] = leaderboardevents.TagAssignmentInfoV1{
			UserID:    entry.UserID,
			TagNumber: entry.TagNumber,
		}
	}

	return results.SuccessResult(payload), nil
}

// GenerateUpdatedSnapshot remains public as it's a useful pure function for testing.
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

	slices.SortFunc(newData, func(a, b leaderboardtypes.LeaderboardEntry) int {
		return int(a.TagNumber - b.TagNumber)
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
