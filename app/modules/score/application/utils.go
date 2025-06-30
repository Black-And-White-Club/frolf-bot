package scoreservice

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

func (s *ScoreService) ProcessScoresForStorage(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, scores []sharedtypes.ScoreInfo) ([]sharedtypes.ScoreInfo, error) {
	if len(scores) == 0 {
		err := fmt.Errorf("cannot process empty score list")
		s.logger.ErrorContext(ctx, "Empty score list provided",
			attr.RoundID("round_id", roundID),
			attr.Error(err),
		)
		s.metrics.RecordOperationFailure(ctx, "ProcessScoresForStorage", roundID)
		return nil, err
	}

	startTime := time.Now()
	s.metrics.RecordOperationAttempt(ctx, "ProcessScoresForStorage", roundID)
	// Optionally, record guildID in logs/metrics if needed

	taggedCount, untaggedCount := 0, 0
	for i := 0; i < len(scores); i++ {
		intScore := int(scores[i].Score)

		// Validate score range for disc golf
		// Using the agreed-upon range of -36 to +72
		if intScore < -36 || intScore > 72 {
			err := fmt.Errorf("invalid score value: %d for user %s. Score must be between -36 and 72", intScore, scores[i].UserID)
			s.logger.WarnContext(ctx, "Invalid score detected during processing",
				attr.RoundID("round_id", roundID),
				attr.String("user_id", string(scores[i].UserID)),
				attr.Int("score", intScore),
				attr.Error(err),
			)
			s.metrics.RecordOperationFailure(ctx, "ProcessScoresForStorage", roundID)
			return nil, err
		}

		scores[i].Score = sharedtypes.Score(intScore)

		s.metrics.RecordPlayerScore(ctx, roundID, scores[i].UserID, scores[i].Score)
		// Optionally, record guildID in logs/metrics if needed

		if scores[i].TagNumber != nil {
			s.metrics.RecordPlayerTag(ctx, roundID, scores[i].UserID, scores[i].TagNumber)
			s.metrics.RecordTagPerformance(ctx, roundID, scores[i].TagNumber, scores[i].Score)
			taggedCount++
		} else {
			s.metrics.RecordUntaggedPlayer(ctx, roundID, scores[i].UserID)
			untaggedCount++
		}
	}

	s.metrics.RecordTaggedPlayersProcessed(ctx, roundID, taggedCount)
	s.metrics.RecordUntaggedPlayersProcessed(ctx, roundID, untaggedCount)

	sortStartTime := time.Now()
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Score < scores[j].Score
	})
	sortDuration := time.Since(sortStartTime)
	s.metrics.RecordScoreSortingDuration(ctx, roundID, sortDuration)

	s.metrics.RecordOperationDuration(ctx, "ProcessScoresForStorage", time.Since(startTime))

	s.logger.InfoContext(ctx, "Scores processed and sorted",
		attr.RoundID("round_id", roundID),
		attr.Int("num_scores", len(scores)),
		attr.Int("tagged_count", taggedCount),
		attr.Int("untagged_count", untaggedCount),
		attr.Float64("sort_duration_seconds", time.Since(sortStartTime).Seconds()),
		attr.Float64("total_duration_seconds", (time.Since(startTime).Seconds())),
	)

	return scores, nil
}
