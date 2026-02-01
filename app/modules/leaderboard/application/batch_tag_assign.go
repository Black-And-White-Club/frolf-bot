package leaderboardservice

import (
	"context"
	"errors"
	"slices"

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
) (results.OperationResult[leaderboardtypes.LeaderboardData, error], error) {

	// Named transaction function for observability
	executeBatchTx := func(ctx context.Context, db bun.IDB) (results.OperationResult[leaderboardtypes.LeaderboardData, error], error) {
		return s.executeBatchLogic(ctx, db, guildID, requests, updateID, source)
	}

	// Wrap with telemetry & transaction
	return withTelemetry(s, ctx, "ExecuteBatchTagAssignment", guildID, func(ctx context.Context) (results.OperationResult[leaderboardtypes.LeaderboardData, error], error) {
		return runInTx(s, ctx, executeBatchTx)
	})
}

// executeBatchLogic contains the core "Funnel" logic.
func (s *LeaderboardService) executeBatchLogic(
	ctx context.Context,
	db bun.IDB,
	guildID sharedtypes.GuildID,
	requests []sharedtypes.TagAssignmentRequest,
	updateID sharedtypes.RoundID,
	source sharedtypes.ServiceUpdateSource,
) (results.OperationResult[leaderboardtypes.LeaderboardData, error], error) {

	s.logger.InfoContext(ctx, "Executing funnel logic",
		attr.String("source", string(source)),
		attr.String("update_id", updateID.String()),
		attr.Int("request_count", len(requests)),
	)

	current, err := s.repo.GetActiveLeaderboard(ctx, db, guildID)
	if err != nil {
		if errors.Is(err, leaderboarddb.ErrNoActiveLeaderboard) {
			current = &leaderboardtypes.Leaderboard{
				LeaderboardData: leaderboardtypes.LeaderboardData{},
			}
		} else {
			return results.OperationResult[leaderboardtypes.LeaderboardData, error]{}, err
		}
	}
	beforeData := current.LeaderboardData

	// --- 2. CONFLICT DETECTION ---
	tagToUserMap := make(map[sharedtypes.TagNumber]sharedtypes.DiscordID)
	userToTagMap := make(map[sharedtypes.DiscordID]sharedtypes.TagNumber)
	for _, entry := range beforeData {
		tagToUserMap[entry.TagNumber] = entry.UserID
		userToTagMap[entry.UserID] = entry.TagNumber
	}

	requestingUsers := make(map[sharedtypes.DiscordID]bool)
	for _, req := range requests {
		requestingUsers[req.UserID] = true
	}

	var conflicts []*TagSwapNeededError

	for _, req := range requests {
		if holderID, occupied := tagToUserMap[req.TagNumber]; occupied {
			if !requestingUsers[holderID] {
				currentTag := userToTagMap[req.UserID]
				conflicts = append(conflicts, &TagSwapNeededError{
					RequestorID:  req.UserID,
					TargetUserID: holderID,
					TargetTag:    req.TagNumber,
					CurrentTag:   currentTag,
				})

				s.logger.InfoContext(ctx, "Tag conflict detected",
					attr.Int("tag", int(req.TagNumber)),
					attr.String("requestor", string(req.UserID)),
					attr.String("current_holder", string(holderID)),
				)
			}
		}
	}

	if len(conflicts) > 0 {
		// Return the first conflict as a domain failure
		return results.FailureResult[leaderboardtypes.LeaderboardData, error](conflicts[0]), nil
	}

	// --- 3. EXECUTION ---
	newData := s.GenerateUpdatedSnapshot(beforeData, requests)

	updatedLB := &leaderboardtypes.Leaderboard{
		GuildID:         guildID,
		LeaderboardData: newData,
		UpdateSource:    source,
		UpdateID:        updateID,
	}

	if err := s.repo.SaveLeaderboard(ctx, db, updatedLB); err != nil {
		return results.OperationResult[leaderboardtypes.LeaderboardData, error]{}, err
	}

	return results.SuccessResult[leaderboardtypes.LeaderboardData, error](updatedLB.LeaderboardData), nil
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
