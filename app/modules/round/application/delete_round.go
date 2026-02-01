package roundservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// ValidateRoundDeletion validates the round delete request.
func (s *RoundService) ValidateRoundDeletion(ctx context.Context, req *roundtypes.DeleteRoundInput) (results.OperationResult[*roundtypes.Round, error], error) {
	return withTelemetry(s, ctx, "ValidateRoundDeletion", req.RoundID, func(ctx context.Context) (results.OperationResult[*roundtypes.Round, error], error) {
		if req.RoundID == sharedtypes.RoundID(uuid.Nil) {
			return results.FailureResult[*roundtypes.Round, error](errors.New("round ID cannot be zero")), nil
		}

		if req.UserID == "" {
			return results.FailureResult[*roundtypes.Round, error](errors.New("requesting user's Discord ID cannot be empty")), nil
		}

		round, err := s.repo.GetRound(ctx, s.db, req.GuildID, req.RoundID)
		if err != nil {
			s.logger.WarnContext(ctx, "Round not found for delete request",
				attr.String("round_id", req.RoundID.String()),
				attr.String("requesting_user", string(req.UserID)),
				attr.Error(err),
			)
			return results.FailureResult[*roundtypes.Round, error](fmt.Errorf("round not found: %w", err)), nil
		}

		if round.CreatedBy != req.UserID {
			s.logger.WarnContext(ctx, "Unauthorized delete request",
				attr.String("round_id", req.RoundID.String()),
				attr.String("requesting_user", string(req.UserID)),
				attr.String("round_created_by", string(round.CreatedBy)),
			)
			return results.FailureResult[*roundtypes.Round, error](errors.New("unauthorized: only the round creator can delete the round")), nil
		}

		s.logger.InfoContext(ctx, "Round delete request validated",
			attr.String("round_id", req.RoundID.String()),
			attr.String("requesting_user", string(req.UserID)),
		)

		return results.SuccessResult[*roundtypes.Round, error](round), nil
	})
}

func (s *RoundService) DeleteRound(ctx context.Context, req *roundtypes.DeleteRoundInput) (results.OperationResult[bool, error], error) {
	deleteOp := func(ctx context.Context, db bun.IDB) (results.OperationResult[bool, error], error) {
		if req.RoundID == sharedtypes.RoundID(uuid.Nil) {
			s.logger.ErrorContext(ctx, "Cannot delete round with nil UUID")
			return results.FailureResult[bool, error](errors.New("round ID cannot be nil")), nil
		}

		s.logger.InfoContext(ctx, "DeleteRound service called",
			attr.RoundID("round_id", req.RoundID),
		)

		// Delete the round from the database
		if err := s.repo.DeleteRound(ctx, db, req.GuildID, req.RoundID); err != nil {
			s.logger.ErrorContext(ctx, "Failed to delete round from DB",
				attr.RoundID("round_id", req.RoundID),
				attr.Error(err),
			)
			return results.FailureResult[bool, error](fmt.Errorf("failed to delete round from database: %w", err)), nil
		}

		s.logger.InfoContext(ctx, "Round deleted from DB", attr.RoundID("round_id", req.RoundID))

		return results.SuccessResult[bool, error](true), nil
	}

	result, err := withTelemetry(s, ctx, "DeleteRound", req.RoundID, func(ctx context.Context) (results.OperationResult[bool, error], error) {
		return runInTx(s, ctx, deleteOp)
	})

	if err != nil {
		return result, err
	}

	if result.IsSuccess() {
		// Attempt to cancel any scheduled jobs for this round (outside transaction, after success)
		if err := s.queueService.CancelRoundJobs(ctx, req.RoundID); err != nil {
			s.logger.WarnContext(ctx, "Failed to cancel scheduled jobs",
				attr.RoundID("round_id", req.RoundID),
				attr.Error(err),
			)
		} else {
			s.logger.InfoContext(ctx, "Scheduled jobs cancellation successful", attr.RoundID("round_id", req.RoundID))
		}
	}

	return result, nil
}
