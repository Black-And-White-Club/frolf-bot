package scoreservice

import (
	"context"
	"fmt"
	"time"

	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// CorrectScore updates a player's score and returns the appropriate payload.
func (s *ScoreService) CorrectScore(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, userID sharedtypes.DiscordID, score sharedtypes.Score, tagNumber *sharedtypes.TagNumber) (ScoreOperationResult, error) {
	return s.serviceWrapper(ctx, "CorrectScore", roundID, func(ctx context.Context) (ScoreOperationResult, error) {
		// Validate the incoming score value at the service layer
		// Adjusted bounds for disc golf: assuming a valid score is between -36 and +72 (e.g., 18 holes, -2 per hole to +4 per hole).
		// Adjust these bounds based on your actual game rules and typical course pars.
		if score < -36 || score > 72 {
			validationError := fmt.Errorf("invalid score value: %v. Score must be between -36 and 72 for disc golf", score)
			s.logger.ErrorContext(ctx, "Invalid score value received",
				attr.ExtractCorrelationID(ctx),
				attr.RoundID("round_id", roundID),
				attr.String("user_id", string(userID)),
				attr.Any("score", score),
				attr.Error(validationError),
			)
			return ScoreOperationResult{
				Failure: &sharedevents.ScoreUpdateFailedPayloadV1{
					GuildID: guildID,
					RoundID: roundID,
					UserID:  userID,
					Reason:  validationError.Error(),
				},
			}, nil // Return nil error as this is a handled business validation failure
		}

		// Preserve existing tag number if not provided
		effectiveTag := tagNumber
		if effectiveTag == nil {
			if existing, err := s.repo.GetScoresForRound(ctx, guildID, roundID); err == nil {
				for _, si := range existing {
					if si.UserID == userID && si.TagNumber != nil {
						tn := *si.TagNumber
						effectiveTag = &tn
						break
					}
				}
			}
		}

		scoreInfo := sharedtypes.ScoreInfo{
			UserID:    userID,
			Score:     score,
			TagNumber: effectiveTag,
		}
		dbStart := time.Now()
		err := s.repo.UpdateOrAddScore(ctx, guildID, roundID, scoreInfo)
		s.metrics.RecordDBQueryDuration(ctx, time.Duration(time.Since(dbStart).Seconds()))
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to update/add score",
				attr.ExtractCorrelationID(ctx),
				attr.RoundID("round_id", roundID),
				attr.String("user_id", string(userID)),
				attr.Error(err),
			)
			// If there's a business error (like record not found),
			// return the Failure payload and a nil error.
			// This signals that the business logic handled the error,
			// and it's not a system error to be propagated further as an 'error'.
			return ScoreOperationResult{
				Failure: &sharedevents.ScoreUpdateFailedPayloadV1{
					GuildID: guildID,
					RoundID: roundID,
					UserID:  userID,
					Reason:  err.Error(),
				},
			}, nil
		}
		s.metrics.RecordScoreCorrectionSuccess(ctx, roundID)
		s.logger.InfoContext(ctx, "Score corrected successfully",
			attr.ExtractCorrelationID(ctx),
			attr.RoundID("round_id", roundID),
			attr.String("user_id", string(userID)),
		)
		return ScoreOperationResult{
			Success: &sharedevents.ScoreUpdatedPayloadV1{
				GuildID: guildID,
				RoundID: roundID,
				UserID:  userID,
				Score:   score,
			},
		}, nil
	})
}
