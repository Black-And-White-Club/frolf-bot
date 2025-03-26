package scoreservice

import (
	"context"
	"time"

	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	scoredb "github.com/Black-And-White-Club/frolf-bot/app/modules/score/infrastructure/repositories"
	"github.com/ThreeDotsLabs/watermill/message"
)

// CorrectScore handles score corrections (manual updates).
func (s *ScoreService) CorrectScore(ctx context.Context, msg *message.Message, event scoreevents.ScoreUpdateRequestPayload) (ScoreOperationResult, error) {
	return s.serviceWrapper(msg, "CorrectScore", event.RoundID, func() (ScoreOperationResult, error) {
		s.metrics.RecordScoreCorrectionAttempt(event.RoundID)

		startTime := time.Now()

		s.logger.Info("Correcting score",
			attr.CorrelationIDFromMsg(msg),
			attr.Int64("round_id", int64(event.RoundID)),
			attr.String("user_id", string(event.UserID)),
			attr.Int("score", int(event.Score)),
		)

		score := &scoredb.Score{
			UserID:    event.UserID,
			RoundID:   event.RoundID,
			Score:     event.Score,
			TagNumber: event.TagNumber,
			Source:    "manual",
		}

		dbStart := time.Now()
		if err := s.ScoreDB.UpdateOrAddScore(ctx, score); err != nil {
			s.metrics.RecordScoreCorrectionFailure(event.RoundID)
			s.logger.Error("Failed to update/add score",
				attr.CorrelationIDFromMsg(msg),
				attr.Int64("round_id", int64(event.RoundID)),
				attr.String("user_id", string(event.UserID)),
				attr.Error(err),
			)
			s.metrics.RecordDBQueryDuration(time.Since(dbStart).Seconds())
			s.metrics.RecordScoreCorrectionDuration(event.RoundID, time.Since(startTime).Seconds())
			return ScoreOperationResult{
				Failure: err,
			}, err
		}
		s.metrics.RecordDBQueryDuration(time.Since(dbStart).Seconds())

		s.metrics.RecordScoreCorrectionSuccess(event.RoundID)
		s.metrics.RecordScoreCorrectionDuration(event.RoundID, time.Since(startTime).Seconds())

		s.logger.Info("Score corrected successfully",
			attr.CorrelationIDFromMsg(msg),
			attr.Int64("round_id", int64(event.RoundID)),
			attr.String("user_id", string(event.UserID)),
			attr.Float64("correction_duration_seconds", time.Since(startTime).Seconds()),
		)

		return ScoreOperationResult{
			Success: score,
		}, nil
	})
}
