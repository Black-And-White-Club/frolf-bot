package roundservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// ValidateScoreUpdateRequest validates the score update request.
func (s *RoundService) ValidateScoreUpdateRequest(ctx context.Context, req *roundtypes.ScoreUpdateRequest) (results.OperationResult[*roundtypes.ScoreUpdateRequest, error], error) {
	result, err := withTelemetry[*roundtypes.ScoreUpdateRequest, error](s, ctx, "ValidateScoreUpdateRequest", req.RoundID, func(ctx context.Context) (results.OperationResult[*roundtypes.ScoreUpdateRequest, error], error) {
		var errs []string
		if req.RoundID == sharedtypes.RoundID(uuid.Nil) {
			errs = append(errs, "round ID cannot be zero")
		}
		if req.UserID == "" {
			errs = append(errs, "participant Discord ID cannot be empty")
		}
		if req.Score == nil {
			errs = append(errs, "score cannot be empty")
		}

		if len(errs) > 0 {
			err := fmt.Errorf("validation errors: %s", strings.Join(errs, "; "))
			s.logger.ErrorContext(ctx, "Score update request validation failed",
				attr.RoundID("round_id", req.RoundID),
				attr.String("guild_id", string(req.GuildID)),
				attr.String("participant", string(req.UserID)),
				attr.Error(err),
			)
			return results.FailureResult[*roundtypes.ScoreUpdateRequest, error](err), nil
		}

		s.logger.InfoContext(ctx, "Score update request validated",
			attr.RoundID("round_id", req.RoundID),
		)

		return results.SuccessResult[*roundtypes.ScoreUpdateRequest, error](req), nil
	})

	return result, err
}

// UpdateParticipantScore updates the participant's score in the database.
func (s *RoundService) UpdateParticipantScore(ctx context.Context, req *roundtypes.ScoreUpdateRequest) (ScoreUpdateResult, error) {
	result, err := withTelemetry[*roundtypes.ScoreUpdateResult, error](s, ctx, "UpdateParticipantScore", req.RoundID, func(ctx context.Context) (ScoreUpdateResult, error) {
		return runInTx[*roundtypes.ScoreUpdateResult, error](s, ctx, func(ctx context.Context, tx bun.IDB) (ScoreUpdateResult, error) {
			// Fetch the round first to get the event message ID
			round, err := s.repo.GetRound(ctx, tx, req.GuildID, req.RoundID)
			if err != nil {
				s.logger.ErrorContext(ctx, "Failed to fetch round",
					attr.RoundID("round_id", req.RoundID),
					attr.String("guild_id", string(req.GuildID)),
					attr.Error(err),
				)
				return results.FailureResult[*roundtypes.ScoreUpdateResult, error](fmt.Errorf("round not found: %w", err)), nil
			}

			// Update the participant's score in the database
			err = s.repo.UpdateParticipantScore(ctx, tx, req.GuildID, req.RoundID, req.UserID, sharedtypes.Score(*req.Score))
			if err != nil {
				s.logger.ErrorContext(ctx, "Failed to update participant score in DB",
					attr.RoundID("round_id", req.RoundID),
					attr.String("guild_id", string(req.GuildID)),
					attr.Error(err),
				)
				return results.FailureResult[*roundtypes.ScoreUpdateResult, error](fmt.Errorf("failed to update score in database: %w", err)), nil
			}

			// Fetch the full, updated list of participants for this round
			updatedParticipants, err := s.repo.GetParticipants(ctx, tx, req.GuildID, req.RoundID)
			if err != nil {
				s.logger.ErrorContext(ctx, "Failed to get updated participants after score update",
					attr.RoundID("round_id", req.RoundID),
					attr.String("guild_id", string(req.GuildID)),
					attr.Error(err),
				)
				return results.FailureResult[*roundtypes.ScoreUpdateResult, error](fmt.Errorf("failed to retrieve updated participants list: %w", err)), nil
			}

			// Return domain result
			return results.SuccessResult[*roundtypes.ScoreUpdateResult, error](&roundtypes.ScoreUpdateResult{
				GuildID:             req.GuildID,
				RoundID:             req.RoundID,
				EventMessageID:      round.EventMessageID,
				UpdatedParticipants: updatedParticipants,
			}), nil
		})
	})

	return result, err
}

// UpdateParticipantScoresBulk updates scores for multiple participants.
func (s *RoundService) UpdateParticipantScoresBulk(ctx context.Context, req *roundtypes.BulkScoreUpdateRequest) (BulkScoreUpdateResult, error) {
	// Implementation pending definition of BulkScoreUpdateRequest and business logic
	return results.SuccessResult[*roundtypes.BulkScoreUpdateResult, error](&roundtypes.BulkScoreUpdateResult{
		GuildID: req.GuildID,
		RoundID: req.RoundID,
		Updates: req.Updates,
	}), nil
}

// CheckAllScoresSubmitted checks if all participants have submitted scores.
func (s *RoundService) CheckAllScoresSubmitted(ctx context.Context, req *roundtypes.CheckAllScoresSubmittedRequest) (AllScoresSubmittedResult, error) {
	result, err := withTelemetry[*roundtypes.AllScoresSubmittedResult, error](s, ctx, "CheckAllScoresSubmitted", req.RoundID, func(ctx context.Context) (AllScoresSubmittedResult, error) {
		participants, err := s.repo.GetParticipants(ctx, nil, req.GuildID, req.RoundID)
		if err != nil {
			return results.FailureResult[*roundtypes.AllScoresSubmittedResult, error](err), nil
		}

		allSubmitted := true
		for _, p := range participants {
			// Only check participants who have accepted the invite
			if p.Response == roundtypes.ResponseAccept && p.Score == nil {
				allSubmitted = false
				break
			}
		}

		var round *roundtypes.Round
		if allSubmitted {
			// Fetch round details if complete
			r, err := s.repo.GetRound(ctx, nil, req.GuildID, req.RoundID)
			if err == nil {
				round = r
			}
		}

		return results.SuccessResult[*roundtypes.AllScoresSubmittedResult, error](&roundtypes.AllScoresSubmittedResult{
			IsComplete:   allSubmitted,
			Participants: participants,
			Round:        round,
		}), nil
	})

	return result, err
}
