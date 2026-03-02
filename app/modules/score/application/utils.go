package scoreservice

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// noTagSortWeight is the sentinel used as a sort key for players without a tag.
// math.MaxInt places untagged players last in ascending-tag sorts.
// Distinct from the -1 sentinel used only for logging.
const noTagSortWeight = math.MaxInt

// computeFinishRanks returns competition-style finish ranks for an already-sorted score slice.
// Tied scores receive the same rank; the subsequent rank is skipped.
// Example: scores [-4, -4, -2] → ranks {alice:1, bob:1, carol:3}.
//
// IMPORTANT: The slice must already be sorted ascending by score using the same comparator
// as ProcessScoresForStorage (score asc, pre-round tag asc for ties). Passing an unsorted
// slice produces incorrect ranks without error. This function is intentionally unexported
// and should only be called immediately after that sort.
func computeFinishRanks(scores []sharedtypes.ScoreInfo) map[sharedtypes.DiscordID]int {
	ranks := make(map[sharedtypes.DiscordID]int, len(scores))
	i := 0
	for i < len(scores) {
		j := i
		for j < len(scores) && scores[j].Score == scores[i].Score {
			j++
		}
		rank := i + 1
		for k := i; k < j; k++ {
			ranks[scores[k].UserID] = rank
		}
		i = j
	}
	return ranks
}

// ProcessScoresForStorage validates, enriches, and sorts scores.
// It returns the sorted scores and the competition-style finish ranks so callers
// do not need to recompute ranks from the same data.
// It is intended to be called within a service transaction.
func (s *ScoreService) ProcessScoresForStorage(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	roundID sharedtypes.RoundID,
	scores []sharedtypes.ScoreInfo,
) ([]sharedtypes.ScoreInfo, map[sharedtypes.DiscordID]int, error) {
	if len(scores) == 0 {
		return nil, nil, fmt.Errorf("%w: cannot process empty score list", ErrInvalidScore)
	}

	startTime := time.Now()
	taggedCount, untaggedCount := 0, 0

	// 1. Validation and Metric Recording
	for i := range scores {
		if scores[i].Score < -36 || scores[i].Score > 72 {
			return nil, nil, fmt.Errorf("%w: %d for user %s (must be between -36 and 72)",
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
			// Tiebreak: lower pre-round tag wins (disc golf convention).
			// Untagged players (nil TagNumber) have lowest priority.
			iTag, jTag := noTagSortWeight, noTagSortWeight
			if scores[i].TagNumber != nil {
				iTag = int(*scores[i].TagNumber)
			}
			if scores[j].TagNumber != nil {
				jTag = int(*scores[j].TagNumber)
			}
			return iTag < jTag
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

	// Compute competition-style finish ranks once; returned to avoid recomputation by the caller.
	finishRanks := computeFinishRanks(scores)

	// Log per-player finish state for observability (tie detection).
	// pre_tag uses -1 as a log-only sentinel for "no tag"; this is distinct from
	// math.MaxInt used by the sort comparator above to place nil-tag players last.
	for _, sc := range scores {
		preTag := -1
		if sc.TagNumber != nil {
			preTag = int(*sc.TagNumber)
		}
		s.logger.InfoContext(ctx, "score processing",
			attr.String("player", string(sc.UserID)),
			attr.Int("score", int(sc.Score)),
			attr.Int("pre_tag", preTag),
			attr.Int("finish_rank", finishRanks[sc.UserID]),
		)
	}

	return scores, finishRanks, nil
}
