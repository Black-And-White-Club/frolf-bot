package roundservice

import (
	"context"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// CancelScheduledRoundStart removes any queued round_start jobs while preserving
// reminder jobs for the round.
func (s *RoundService) CancelScheduledRoundStart(ctx context.Context, roundID sharedtypes.RoundID) error {
	if s.queueService == nil {
		return nil
	}

	return s.queueService.CancelRoundStartJobs(ctx, roundID)
}
