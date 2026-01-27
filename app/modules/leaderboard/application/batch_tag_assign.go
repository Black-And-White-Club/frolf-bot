package leaderboardservice

import (
	"context"
	"database/sql"
	"errors"
	"slices"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
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
) (leaderboardtypes.LeaderboardData, error) {

	// Run within a DB transaction if we have a DB configured, otherwise execute directly.
	if s.db == nil {
		return s.executeBatchLogic(ctx, nil, guildID, requests, updateID, source)
	}

	var result leaderboardtypes.LeaderboardData
	err := s.db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		var innerErr error
		result, innerErr = s.executeBatchLogic(ctx, tx, guildID, requests, updateID, source)
		return innerErr
	})

	return result, err
}

// executeBatchLogic contains the core "Funnel" logic.
func (s *LeaderboardService) executeBatchLogic(
	ctx context.Context,
	db bun.IDB,
	guildID sharedtypes.GuildID,
	requests []sharedtypes.TagAssignmentRequest,
	updateID sharedtypes.RoundID,
	source sharedtypes.ServiceUpdateSource,
) (leaderboardtypes.LeaderboardData, error) {

	s.logger.InfoContext(ctx, "Executing funnel logic",
		attr.String("source", string(source)),
		attr.String("update_id", updateID.String()),
		attr.Int("request_count", len(requests)),
	)

	current, err := s.repo.GetActiveLeaderboard(ctx, db, guildID)
	if err != nil {
		if errors.Is(err, leaderboarddb.ErrNoActiveLeaderboard) {
			current = &leaderboarddb.Leaderboard{
				LeaderboardData: leaderboardtypes.LeaderboardData{},
			}
		} else {
			return nil, err
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
		// Optionally, return the first conflict (compatible with current handler)
		return nil, conflicts[0]
	}

	// --- 3. EXECUTION ---
	newData := s.GenerateUpdatedSnapshot(beforeData, requests)

	updatedLB, err := s.repo.UpdateLeaderboard(ctx, db, guildID, newData, updateID, source)
	if err != nil {
		return nil, err
	}

	return updatedLB.LeaderboardData, nil
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
