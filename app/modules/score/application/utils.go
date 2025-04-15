package scoreservice

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// ProcessScoresForStorage prepares score data for storage by converting and sorting.
// It returns the processed score info sorted by score (lowest first).
func (s *ScoreService) ProcessScoresForStorage(
	ctx context.Context,
	roundID sharedtypes.RoundID,
	scores []sharedtypes.ScoreInfo,
) ([]sharedtypes.ScoreInfo, error) {
	// Quick validation
	if len(scores) == 0 {
		err := fmt.Errorf("cannot process empty score list")
		s.logger.ErrorContext(ctx, "Empty score list provided",
			attr.RoundID("round_id", roundID),
			attr.Error(err),
		)
		s.metrics.RecordOperationFailure("ProcessScoresForStorage", roundID)
		return nil, err
	}

	// Start timing and metrics
	startTime := time.Now()
	s.metrics.RecordOperationAttempt("ProcessScoresForStorage", roundID)

	// Process scores
	taggedCount, untaggedCount := 0, 0
	for i := 0; i < len(scores); i++ {
		// Normalize score inline for better performance
		intScore := int(scores[i].Score)

		// Bounds checking only for extreme scores (moved inline from normalizeScore)
		if intScore < -100 || intScore > 100 {
			s.logger.Debug("Unusually extreme score detected", attr.Int("score", intScore))
		}

		scores[i].Score = sharedtypes.Score(intScore)

		// Track metrics
		s.metrics.RecordPlayerScore(roundID, scores[i].UserID, scores[i].Score)

		// Count tagged vs untagged
		if scores[i].TagNumber != nil {
			s.metrics.RecordPlayerTag(roundID, scores[i].UserID, scores[i].TagNumber)
			s.metrics.RecordTagPerformance(roundID, scores[i].TagNumber, scores[i].Score)
			taggedCount++
		} else {
			s.metrics.RecordUntaggedPlayer(roundID, scores[i].UserID)
			untaggedCount++
		}
	}

	// Record tagged vs untagged counts
	s.metrics.RecordTaggedPlayersProcessed(roundID, taggedCount)
	s.metrics.RecordUntaggedPlayersProcessed(roundID, untaggedCount)

	// Sort scores
	sortStartTime := time.Now()
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Score < scores[j].Score
	})
	sortDuration := time.Since(sortStartTime).Seconds()
	s.metrics.RecordScoreSortingDuration(roundID, sortDuration)

	// Log completion and record metrics
	s.metrics.RecordOperationDuration("ProcessScoresForStorage", time.Since(startTime).Seconds())

	s.logger.InfoContext(ctx, "Scores processed and sorted",
		attr.RoundID("round_id", roundID),
		attr.Int("num_scores", len(scores)),
		attr.Int("tagged_count", taggedCount),
		attr.Int("untagged_count", untaggedCount),
		attr.Float64("sort_duration_seconds", sortDuration),
		attr.Float64("total_duration_seconds", time.Since(startTime).Seconds()),
	)

	return scores, nil
}

// ExtractTagInformation extracts only users with tags from the processed scores.
// This creates a format suitable for the leaderboard update.
// func (s *ScoreService) ExtractTagInformation(
// 	ctx context.Context,
// 	roundID sharedtypes.RoundID,
// 	processedScores []sharedtypes.ScoreInfo,
// ) ([]byte, error) {
// 	operationName := "ExtractTagInformation"

// 	s.logger.InfoContext(ctx,"Starting "+operationName,
// 		attr.RoundID("round_id", roundID),
// 		attr.Int("num_scores", len(processedScores)),
// 	)

// 	// Record metrics
// 	s.metrics.RecordOperationAttempt(operationName, roundID)
// 	startTime := time.Now()
// 	defer func() {
// 		duration := time.Since(startTime).Seconds()
// 		s.metrics.RecordOperationDuration(operationName, duration)
// 	}()

// 	// Create a slice to hold participant tags
// 	participantTags := make([]string, 0, len(processedScores))

// 	for _, scoreInfo := range processedScores {
// 		// Format the tag information as needed
// 		tagInfo := fmt.Sprintf("User  :%s:Tag:%d", scoreInfo.UserID, scoreInfo.TagNumber)
// 		participantTags = append(participantTags, tagInfo)

// 		// Record individual player tag
// 		s.metrics.RecordPlayerTag(roundID, scoreInfo.UserID, scoreInfo.TagNumber)
// 	}

// 	s.logger.InfoContext(ctx,"Tag information extracted",
// 		attr.RoundID("round_id", roundID),
// 		attr.Int("num_participant_tags", len(participantTags)),
// 	)

// 	// Marshal the participantTags slice to JSON
// 	jsonData, err := json.Marshal(participantTags)
// 	if err != nil {
// 		s.logger.ErrorContext(ctx,"Failed to marshal tag information to JSON",
// 			attr.RoundID("round_id", roundID),
// 			attr.Error(err),
// 		)
// 		return nil, err
// 	}

// 	return jsonData, nil
// }
