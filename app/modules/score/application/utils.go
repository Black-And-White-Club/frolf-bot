package scoreservice

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// ProcessScoresForStorage validates, enriches, and sorts scores.
// It is intended to be called within a service transaction.
func (s *ScoreService) ProcessScoresForStorage(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	roundID sharedtypes.RoundID,
	scores []sharedtypes.ScoreInfo,
) ([]sharedtypes.ScoreInfo, error) {
	if len(scores) == 0 {
		return nil, fmt.Errorf("%w: cannot process empty score list", ErrInvalidScore)
	}

	startTime := time.Now()
	taggedCount, untaggedCount := 0, 0

	// 1. Validation and Metric Recording
	for i := range scores {
		if scores[i].Score < -36 || scores[i].Score > 72 {
			return nil, fmt.Errorf("%w: %d for user %s (must be between -36 and 72)",
				ErrInvalidScore, scores[i].Score, scores[i].UserID)
		}

		// Record Individual Metrics
		s.metrics.RecordPlayerScore(ctx, roundID, scores[i].UserID, scores[i].Score)

		if scores[i].TagNumber != nil {
			s.metrics.RecordPlayerTag(ctx, roundID, scores[i].UserID, scores[i].TagNumber)
			s.metrics.RecordTagPerformance(ctx, roundID, scores[i].TagNumber, scores[i].Score)
			taggedCount++
		} else {
			s.metrics.RecordUntaggedPlayer(ctx, roundID, scores[i].UserID)
			untaggedCount++
		}
	}

	// 2. Sorting Logic
	sortStartTime := time.Now()
	sort.Slice(scores, func(i, j int) bool {
		if scores[i].Score == scores[j].Score {
			// Deterministic tie-breaking by UserID if scores are equal
			return scores[i].UserID < scores[j].UserID
		}
		return scores[i].Score < scores[j].Score
	})

	// 3. Batch Metrics & Logging
	s.metrics.RecordScoreSortingDuration(ctx, roundID, time.Since(sortStartTime))
	s.metrics.RecordTaggedPlayersProcessed(ctx, roundID, taggedCount)
	s.metrics.RecordUntaggedPlayersProcessed(ctx, roundID, untaggedCount)
	s.metrics.RecordOperationDuration(ctx, "ProcessScoresForStorage", time.Since(startTime))

	s.logger.InfoContext(ctx, "Scores processed and sorted",
		attr.RoundID("round_id", roundID),
		attr.Int("num_scores", len(scores)),
		attr.Int("tagged_count", taggedCount),
		attr.Duration("total_duration", time.Since(startTime)),
	)

	return scores, nil
}
