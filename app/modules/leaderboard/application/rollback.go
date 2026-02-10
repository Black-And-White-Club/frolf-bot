package leaderboardservice

import (
	"context"
	"fmt"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/uptrace/bun"
)

// rollbackRoundPoints reverses the point effects of a previous processing of this round.
// It subtracts points from SeasonStandings and deletes the PointHistory.
// This ensures idempotency: running ProcessRound multiple times won't double-count points.
func (s *LeaderboardService) rollbackRoundPoints(ctx context.Context, tx bun.IDB, roundID sharedtypes.RoundID) error {
	// 1. Fetch exactly what we awarded previously
	//    We need the MemberID and the Points amount to reverse it accurately.
	history, err := s.repo.GetPointHistoryForRound(ctx, tx, roundID)
	if err != nil {
		return fmt.Errorf("failed to fetch point history for rollback: %w", err)
	}

	// If no history exists, this is a fresh round (or already rolled back). Do nothing.
	if len(history) == 0 {
		return nil
	}

	s.logger.Info("Rolling back points for round", "round_id", roundID, "entries", len(history))

	// 2. Prepare the rollback updates
	//    We loop through history and subtract the points from the standing.
	for _, h := range history {
		// We subtract the points and decrement the rounds played count.
		err := s.repo.DecrementSeasonStanding(
			ctx,
			tx,
			h.MemberID,
			h.Points, // Amount to remove
		)
		if err != nil {
			return fmt.Errorf("failed to rollback standing for member %s: %w", h.MemberID, err)
		}
	}

	// 3. Clear the history log
	//    This ensures that when we run the "Forward" pass next, we don't duplicate history.
	if err := s.repo.DeletePointHistoryForRound(ctx, tx, roundID); err != nil {
		return fmt.Errorf("failed to delete point history for round %s: %w", roundID, err)
	}

	return nil
}
