package leaderboardservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
)

// -- Leaderboard Update --

// UpdateLeaderboard updates the leaderboard based on the provided round ID and sorted participant tags.
func (s *LeaderboardService) UpdateLeaderboard(ctx context.Context, roundID sharedtypes.RoundID, sortedParticipantTags []string) (LeaderboardOperationResult, error) {
	s.metrics.RecordLeaderboardUpdateAttempt(ctx, roundID, "LeaderboardService")
	correlationID := attr.ExtractCorrelationID(ctx)
	roundIDAttr := attr.RoundID("round_id", roundID)

	s.logger.InfoContext(ctx, "Leaderboard update triggered", correlationID, roundIDAttr)

	handleError := func(reason string, err error) (LeaderboardOperationResult, error) {
		if err == nil {
			err = errors.New(reason)
		} else {
			err = fmt.Errorf("%s: %w", reason, err)
		}

		s.logger.ErrorContext(ctx, reason, correlationID, roundIDAttr, attr.Error(err))
		s.metrics.RecordLeaderboardUpdateFailure(ctx, roundID, "LeaderboardService")

		return LeaderboardOperationResult{
			Failure: &leaderboardevents.LeaderboardUpdateFailedPayload{
				RoundID: roundID,
				Reason:  reason,
			},
			Error: err,
		}, err
	}

	if len(sortedParticipantTags) == 0 {
		return handleError("invalid input: empty sorted participant tags", nil)
	}

	return s.serviceWrapper(ctx, "UpdateLeaderboard", func(ctx context.Context) (LeaderboardOperationResult, error) {
		dbStartTime := time.Now()
		currentLeaderboard, err := s.LeaderboardDB.GetActiveLeaderboard(ctx)
		s.metrics.RecordOperationDuration(ctx, "GetActiveLeaderboard", "LeaderboardService", time.Since(dbStartTime))

		if err != nil {
			// Handle sql.ErrNoRows specifically if necessary, but a general DB error is also fine
			return handleError("database connection error", err)
		}

		// If no active leaderboard exists, start with empty data
		if currentLeaderboard == nil {
			currentLeaderboard = &leaderboarddb.Leaderboard{
				LeaderboardData: []leaderboardtypes.LeaderboardEntry{},
			}
		}

		updatedLeaderboardData, err := s.GenerateUpdatedLeaderboard(currentLeaderboard.LeaderboardData, sortedParticipantTags)
		if err != nil {
			return handleError(fmt.Sprintf("failed to generate updated leaderboard data: %v", err), err)
		}

		// This check might be redundant if GenerateUpdatedLeaderboard guarantees non-empty output
		// when sortedParticipantTags is non-empty, but keeping it for safety.
		if len(updatedLeaderboardData) == 0 {
			return handleError("invalid generated leaderboard data: empty data slice", nil)
		}

		dbStartTime = time.Now()
		err = s.LeaderboardDB.UpdateLeaderboard(ctx, updatedLeaderboardData, roundID)
		s.metrics.RecordOperationDuration(ctx, "UpdateLeaderboard", "LeaderboardService", time.Since(dbStartTime))

		if err != nil {
			return handleError("failed to update leaderboard in database", err)
		}

		s.logger.InfoContext(ctx, "Leaderboard updated successfully", correlationID, roundIDAttr)

		return LeaderboardOperationResult{
			Success: &leaderboardevents.LeaderboardUpdatedPayload{
				RoundID: roundID,
			},
		}, nil
	})
}
