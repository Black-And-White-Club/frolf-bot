package leaderboardservice

import (
	"context"
	"errors"
	"time"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// -- Leaderboard Update --

// UpdateLeaderboard updates the leaderboard based on the provided round ID and sorted participant tags.
func (s *LeaderboardService) UpdateLeaderboard(ctx context.Context, roundID sharedtypes.RoundID, sortedParticipantTags []string) (LeaderboardOperationResult, error) {
	// Record attempt once
	s.metrics.RecordLeaderboardUpdateAttempt(roundID, "LeaderboardService")
	correlationID := attr.ExtractCorrelationID(ctx)
	roundIDAttr := attr.RoundID("round_id", roundID)

	// Single log for operation start
	s.logger.InfoContext(ctx, "Leaderboard update triggered", correlationID, roundIDAttr)

	// Common error handler to avoid code duplication
	handleError := func(reason string, err error) (LeaderboardOperationResult, error) {
		if err == nil {
			err = errors.New(reason)
		}

		s.logger.ErrorContext(ctx, reason, correlationID, roundIDAttr, attr.Error(err))
		s.metrics.RecordLeaderboardUpdateFailure(roundID, "LeaderboardService")

		return LeaderboardOperationResult{
			Failure: &leaderboardevents.LeaderboardUpdateFailedPayload{
				RoundID: roundID,
				Reason:  reason,
			},
			Error: err,
		}, err
	}

	// Early validation
	if len(sortedParticipantTags) == 0 {
		return handleError("invalid input: empty sorted participant tags", nil)
	}

	return s.serviceWrapper(ctx, "UpdateLeaderboard", func() (LeaderboardOperationResult, error) {
		// 1. Get the current active leaderboard
		dbStartTime := time.Now()
		currentLeaderboard, err := s.LeaderboardDB.GetActiveLeaderboard(ctx)
		s.metrics.RecordOperationDuration("GetActiveLeaderboard", "LeaderboardService", time.Since(dbStartTime).Seconds())

		if err != nil {
			return handleError("database connection error", err)
		}

		if currentLeaderboard == nil || len(currentLeaderboard.LeaderboardData) == 0 {
			return handleError("invalid leaderboard data", nil)
		}

		// 2. Generate the updated leaderboard
		updatedLeaderboard, err := s.GenerateUpdatedLeaderboard(currentLeaderboard, sortedParticipantTags)
		if err != nil {
			return handleError("failed to generate updated leaderboard", err)
		}

		if updatedLeaderboard == nil || len(updatedLeaderboard.LeaderboardData) == 0 {
			return handleError("invalid generated leaderboard data", nil)
		}

		// 3. Update the leaderboard in the database
		dbStartTime = time.Now()
		err = s.LeaderboardDB.UpdateLeaderboard(ctx, updatedLeaderboard.LeaderboardData, roundID)
		s.metrics.RecordOperationDuration("UpdateLeaderboard", "LeaderboardService", time.Since(dbStartTime).Seconds())

		if err != nil {
			return handleError("failed to update leaderboard", err)
		}

		// 4. Return success result
		s.logger.InfoContext(ctx, "Leaderboard updated successfully", correlationID, roundIDAttr)
		return LeaderboardOperationResult{
			Success: &leaderboardevents.LeaderboardUpdatedPayload{
				RoundID: roundID,
			},
		}, nil
	})
}
