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
		scoreInfo := sharedtypes.ScoreInfo{
			UserID:    userID,
			Score:     score,
			TagNumber: tagNumber,
		}
		dbStart := time.Now()
		err := s.ScoreDB.UpdateOrAddScore(ctx, roundID, scoreInfo)
		s.metrics.RecordDBQueryDuration(ctx, time.Duration(time.Since(dbStart).Seconds()))
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to update/add score",
				attr.ExtractCorrelationID(ctx),
				attr.RoundID("round_id", roundID),
				attr.String("user_id", string(userID)),
				attr.Error(err),
			)
			return ScoreOperationResult{
				Failure: &scoreevents.ScoreUpdateFailurePayload{
					RoundID: roundID,
					UserID:  userID,
					Error:   err.Error(), // Use the error message from the DB.
				},
				Error: err,
			}, err
		}
		s.metrics.RecordScoreCorrectionSuccess(ctx, roundID)
		s.logger.InfoContext(ctx, "Score corrected successfully",
			attr.ExtractCorrelationID(ctx),
			attr.RoundID("round_id", roundID),
			attr.String("user_id", string(userID)),
		)
		return ScoreOperationResult{
			Success: &scoreevents.ScoreUpdateSuccessPayload{
				RoundID: roundID,
				UserID:  userID,
				Score:   score,
			},
		}, nil
	})
}
