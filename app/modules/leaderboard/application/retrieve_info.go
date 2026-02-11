package leaderboardservice

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/uptrace/bun"
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
) (results.OperationResult[[]leaderboardtypes.LeaderboardEntry, error], error) {

	return withTelemetry(s, ctx, "GetLeaderboard", guildID, func(ctx context.Context) (results.OperationResult[[]leaderboardtypes.LeaderboardEntry, error], error) {
		getLeaderboardTx := func(ctx context.Context, db bun.IDB) (results.OperationResult[[]leaderboardtypes.LeaderboardEntry, error], error) {
			leaderboard, err := s.repo.GetActiveLeaderboard(ctx, db, guildID)
			if err != nil {
				return results.OperationResult[[]leaderboardtypes.LeaderboardEntry, error]{}, err
			}
			if leaderboard == nil {
				// This case might be unreachable if repo returns error for no active leaderboard, but keeping for safety
				return results.FailureResult[[]leaderboardtypes.LeaderboardEntry, error](leaderboarddb.ErrNoActiveLeaderboard), nil
			}

			// Return a copy of entries
			entries := make([]leaderboardtypes.LeaderboardEntry, len(leaderboard.LeaderboardData))
			copy(entries, leaderboard.LeaderboardData)

			// Enrich entries with season standings (TotalPoints, RoundsPlayed)
			userIDs := make([]sharedtypes.DiscordID, len(entries))
			for i, e := range entries {
				userIDs[i] = e.UserID
			}
			standings, err := s.repo.GetSeasonStandings(ctx, db, "", userIDs)
			if err != nil {
				s.logger.ErrorContext(ctx, "failed to enrich leaderboard with season standings", attr.Error(err))
			} else {
				for i := range entries {
					if st, ok := standings[entries[i].UserID]; ok {
						entries[i].TotalPoints = st.TotalPoints
						entries[i].RoundsPlayed = st.RoundsPlayed
					}
				}
			}

			return results.SuccessResult[[]leaderboardtypes.LeaderboardEntry, error](entries), nil
		}

		return runInTx(s, ctx, getLeaderboardTx)
	})
}

// GetTagByUserID returns the tag number for a given user.
func (s *LeaderboardService) GetTagByUserID(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	userID sharedtypes.DiscordID,
) (results.OperationResult[sharedtypes.TagNumber, error], error) {

	return withTelemetry(s, ctx, "GetTagByUserID", guildID, func(ctx context.Context) (results.OperationResult[sharedtypes.TagNumber, error], error) {
		leaderboard, err := s.repo.GetActiveLeaderboard(ctx, s.db, guildID)
		if err != nil {
			return results.OperationResult[sharedtypes.TagNumber, error]{}, err
		}

		for _, entry := range leaderboard.LeaderboardData {
			if entry.UserID == userID {
				return results.SuccessResult[sharedtypes.TagNumber, error](entry.TagNumber), nil
			}
		}

		return results.FailureResult[sharedtypes.TagNumber, error](sql.ErrNoRows), nil
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
		leaderboard, err := s.repo.GetActiveLeaderboard(ctx, s.db, guildID)
		if err != nil {
			if errors.Is(err, leaderboarddb.ErrNoActiveLeaderboard) {
				return results.SuccessResult[TagAvailabilityResult, error](TagAvailabilityResult{Available: false, Reason: "no active leaderboard"}), nil
			}
			return results.OperationResult[TagAvailabilityResult, error]{}, err
		}

		available, reason := checkInternalAvailability(leaderboard, userID, tagNumber)
		return results.SuccessResult[TagAvailabilityResult, error](TagAvailabilityResult{Available: available, Reason: reason}), nil
	})
}

// Private helper function
func checkInternalAvailability(l *leaderboardtypes.Leaderboard, userID sharedtypes.DiscordID, tag sharedtypes.TagNumber) (bool, string) {
	for _, entry := range l.LeaderboardData {
		if entry.TagNumber == tag && entry.UserID != userID {
			return false, "tag is already taken"
		}
	}
	return true, ""
}
