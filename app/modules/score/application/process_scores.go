package scoreservice

import (
	"context"
	"sort"
	"time"

	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	scoredb "github.com/Black-And-White-Club/frolf-bot/app/modules/score/infrastructure/repositories"
	"github.com/ThreeDotsLabs/watermill/message"
)

// ProcessRoundScores processes scores received from the round module.
func (s *ScoreService) ProcessRoundScores(ctx context.Context, msg *message.Message, event scoreevents.ProcessRoundScoresRequestPayload) (ScoreOperationResult, error) {
	return s.serviceWrapper(msg, "ProcessRoundScores", event.RoundID, func() (ScoreOperationResult, error) {
		s.metrics.RecordScoreProcessingAttempt(event.RoundID)

		startTime := time.Now()

		s.logger.Info("Processing round scores",
			attr.CorrelationIDFromMsg(msg),
			attr.Int64("round_id", int64(event.RoundID)),
			attr.Int("num_scores", len(event.Scores)),
		)

		var scores []scoredb.Score
		for _, score := range event.Scores {
			scores = append(scores, scoredb.Score{
				UserID:    score.UserID,
				Score:     score.Score,
				TagNumber: &score.TagNumber,
			})
		}

		sort.Slice(scores, func(i, j int) bool {
			return scores[i].Score < scores[j].Score
		})

		s.logger.Info("Sorted round scores",
			attr.CorrelationIDFromMsg(msg),
			attr.Int64("round_id", int64(event.RoundID)),
			attr.Int("num_scores", len(scores)),
		)

		defer func() {
			if r := recover(); r != nil {
				s.metrics.RecordScoreProcessingFailure(event.RoundID)
				s.logger.Error("Panic occurred during score processing",
					attr.CorrelationIDFromMsg(msg),
					attr.Int64("round_id", int64(event.RoundID)),
					attr.Any("panic", r),
				)
				s.metrics.RecordScoreProcessingDuration(event.RoundID, time.Since(startTime).Seconds())
			}
		}()

		dbStart := time.Now()
		if err := s.ScoreDB.LogScores(ctx, event.RoundID, scores, "auto"); err != nil {
			s.metrics.RecordScoreProcessingFailure(event.RoundID)
			s.logger.Error("Failed to log scores to database",
				attr.CorrelationIDFromMsg(msg),
				attr.Int64("round_id", int64(event.RoundID)),
				attr.Error(err),
			)
			s.metrics.RecordDBQueryDuration(time.Since(dbStart).Seconds())
			s.metrics.RecordScoreProcessingDuration(event.RoundID, time.Since(startTime).Seconds())
			return ScoreOperationResult{
				Failure: err,
			}, err
		}
		s.metrics.RecordDBQueryDuration(time.Since(dbStart).Seconds())

		s.metrics.RecordScoreProcessingSuccess(event.RoundID)
		s.metrics.RecordScoreProcessingDuration(event.RoundID, time.Since(startTime).Seconds())

		s.logger.Info("Round scores processed successfully",
			attr.CorrelationIDFromMsg(msg),
			attr.Int64("round_id", int64(event.RoundID)),
			attr.Float64("processing_duration_seconds", time.Since(startTime).Seconds()),
		)

		return ScoreOperationResult{
			Success: scores,
		}, nil
	})
}
