package scoreservice

import (
	"context"
	"time"

	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// CorrectScore updates a player's score and returns the appropriate payload.
func (s *ScoreService) CorrectScore(ctx context.Context, roundID sharedtypes.RoundID, userID sharedtypes.DiscordID, score sharedtypes.Score, tagNumber *sharedtypes.TagNumber) (ScoreOperationResult, error) {
	return s.serviceWrapper(ctx, "CorrectScore", roundID, func(ctx context.Context) (ScoreOperationResult, error) {
		// Extract correlation ID for logging
		correlationID := attr.ExtractCorrelationID(ctx)

		// Start tracing span
		ctx, span := s.tracer.StartSpan(ctx, "CorrectScore.DatabaseOperation", nil)
		defer span.End()

		// Prepare the score info
		scoreInfo := sharedtypes.ScoreInfo{
			UserID:    userID,
			Score:     score,
			TagNumber: tagNumber,
		}

		// Attempt to update score in the database
		dbStart := time.Now()
		err := s.ScoreDB.UpdateOrAddScore(ctx, roundID, scoreInfo)
		s.metrics.RecordDBQueryDuration(time.Since(dbStart).Seconds())

		if err != nil {
			// Log error
			s.logger.Error("Failed to update/add score",
				attr.LogAttr(correlationID),
				attr.RoundID("round_id", roundID),
				attr.String("user_id", string(userID)),
				attr.Error(err),
			)

			// Return failure payload
			return ScoreOperationResult{
				Failure: &scoreevents.ScoreUpdateFailurePayload{
					RoundID: roundID,
					UserID:  userID,
					Error:   err.Error(),
				},
				Error: err,
			}, err
		}

		// Record successful score correction
		s.metrics.RecordScoreCorrectionSuccess(roundID)

		// Log success
		s.logger.Info("Score corrected successfully",
			attr.LogAttr(correlationID),
			attr.RoundID("round_id", roundID),
			attr.String("user_id", string(userID)),
		)

		// Return success payload
		return ScoreOperationResult{
			Success: &scoreevents.ScoreUpdateSuccessPayload{
				RoundID: roundID,
				UserID:  userID,
				Score:   score,
			},
		}, nil
	})
}
