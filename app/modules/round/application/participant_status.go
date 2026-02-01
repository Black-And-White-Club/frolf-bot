package roundservice

import (
	"context"
	"fmt"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// CheckParticipantStatus checks if a join request is a toggle or requires validation.
func (s *RoundService) CheckParticipantStatus(ctx context.Context, req *roundtypes.JoinRoundRequest) (results.OperationResult[*roundtypes.ParticipantStatusCheckResult, error], error) {
	result, err := withTelemetry[*roundtypes.ParticipantStatusCheckResult, error](s, ctx, "CheckParticipantStatus", req.RoundID, func(ctx context.Context) (results.OperationResult[*roundtypes.ParticipantStatusCheckResult, error], error) {
		s.logger.InfoContext(ctx, "Checking participant status",
			attr.RoundID("round_id", req.RoundID),
			attr.String("user_id", string(req.UserID)),
			attr.String("requested_response", string(req.Response)),
		)

		// Check if the user is already a participant
		participant, err := s.repo.GetParticipant(ctx, nil, req.GuildID, req.RoundID, req.UserID)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to get participant's current status",
				attr.RoundID("round_id", req.RoundID),
				attr.String("user_id", string(req.UserID)),
				attr.Error(err),
			)
			return results.FailureResult[*roundtypes.ParticipantStatusCheckResult, error](fmt.Errorf("failed to get participant status: %w", err)), nil
		}

		currentStatus := ""
		if participant != nil {
			currentStatus = string(participant.Response)
		}

		action := "VALIDATE"
		if currentStatus == string(req.Response) {
			action = "REMOVE"
		}

		s.logger.InfoContext(ctx, "Participant status checked",
			attr.RoundID("round_id", req.RoundID),
			attr.String("user_id", string(req.UserID)),
			attr.String("current_status", currentStatus),
			attr.String("action", action),
		)

		return results.SuccessResult[*roundtypes.ParticipantStatusCheckResult, error](&roundtypes.ParticipantStatusCheckResult{
			Action:        action,
			CurrentStatus: currentStatus,
			RoundID:       req.RoundID,
			UserID:        req.UserID,
			Response:      req.Response,
			GuildID:       req.GuildID,
		}), nil
	})

	return result, err
}

// ValidateJoinRequest validates the basic details of a join request.
func (s *RoundService) ValidateJoinRequest(ctx context.Context, req *roundtypes.JoinRoundRequest) (results.OperationResult[*roundtypes.JoinRoundRequest, error], error) {
	result, err := withTelemetry[*roundtypes.JoinRoundRequest, error](s, ctx, "ValidateJoinRequest", req.RoundID, func(ctx context.Context) (results.OperationResult[*roundtypes.JoinRoundRequest, error], error) {
		var errorMessages []string
		if req.RoundID == sharedtypes.RoundID(uuid.Nil) {
			errorMessages = append(errorMessages, "round ID cannot be nil")
		}
		if req.UserID == "" {
			errorMessages = append(errorMessages, "participant Discord ID cannot be empty")
		}

		if len(errorMessages) > 0 {
			return results.FailureResult[*roundtypes.JoinRoundRequest, error](fmt.Errorf("validation failed: %v", errorMessages)), nil
		}

		// Determine if Join is Late
		round, err := s.repo.GetRound(ctx, nil, req.GuildID, req.RoundID)
		if err != nil {
			return results.FailureResult[*roundtypes.JoinRoundRequest, error](fmt.Errorf("failed to fetch round details: %w", err)), nil
		}

		isLateJoin := round.State == roundtypes.RoundStateInProgress || round.State == roundtypes.RoundStateFinalized
		req.JoinedLate = &isLateJoin

		return results.SuccessResult[*roundtypes.JoinRoundRequest, error](req), nil
	})

	return result, err
}

// UpdateParticipantStatus handles the actual joining/updating of the round participant.
func (s *RoundService) UpdateParticipantStatus(ctx context.Context, req *roundtypes.JoinRoundRequest) (results.OperationResult[*roundtypes.Round, error], error) {
	result, err := withTelemetry[*roundtypes.Round, error](s, ctx, "UpdateParticipantStatus", req.RoundID, func(ctx context.Context) (results.OperationResult[*roundtypes.Round, error], error) {
		return runInTx[*roundtypes.Round, error](s, ctx, func(ctx context.Context, tx bun.IDB) (results.OperationResult[*roundtypes.Round, error], error) {
			participant := roundtypes.Participant{
				UserID:    req.UserID,
				Response:  req.Response,
				TagNumber: req.TagNumber,
				Score:     nil,
			}

			// Update participant in DB
			_, err := s.repo.UpdateParticipant(ctx, tx, req.GuildID, req.RoundID, participant)
			if err != nil {
				return results.FailureResult[*roundtypes.Round, error](fmt.Errorf("failed to update participant in DB: %w", err)), nil
			}

			// Fetch updated round to return
			round, err := s.repo.GetRound(ctx, tx, req.GuildID, req.RoundID)
			if err != nil {
				return results.FailureResult[*roundtypes.Round, error](fmt.Errorf("failed to fetch updated round: %w", err)), nil
			}

			return results.SuccessResult[*roundtypes.Round, error](round), nil
		})
	})

	return result, err
}

// JoinRound is a wrapper for UpdateParticipantStatus to satisfy interface if needed.
func (s *RoundService) JoinRound(ctx context.Context, req *roundtypes.JoinRoundRequest) (results.OperationResult[*roundtypes.Round, error], error) {
	return s.UpdateParticipantStatus(ctx, req)
}

// ParticipantRemoval handles removing a participant from a round.
func (s *RoundService) ParticipantRemoval(ctx context.Context, req *roundtypes.JoinRoundRequest) (results.OperationResult[*roundtypes.Round, error], error) {
	result, err := withTelemetry[*roundtypes.Round, error](s, ctx, "ParticipantRemoval", req.RoundID, func(ctx context.Context) (results.OperationResult[*roundtypes.Round, error], error) {
		return runInTx[*roundtypes.Round, error](s, ctx, func(ctx context.Context, tx bun.IDB) (results.OperationResult[*roundtypes.Round, error], error) {
			_, err := s.repo.RemoveParticipant(ctx, tx, req.GuildID, req.RoundID, req.UserID)
			if err != nil {
				return results.FailureResult[*roundtypes.Round, error](fmt.Errorf("failed to remove participant: %w", err)), nil
			}

			round, err := s.repo.GetRound(ctx, tx, req.GuildID, req.RoundID)
			if err != nil {
				return results.FailureResult[*roundtypes.Round, error](fmt.Errorf("failed to fetch updated round: %w", err)), nil
			}

			return results.SuccessResult[*roundtypes.Round, error](round), nil
		})
	})

	return result, err
}

// ValidateParticipantJoinRequest is a pass-through to ValidateJoinRequest
func (s *RoundService) ValidateParticipantJoinRequest(ctx context.Context, req *roundtypes.JoinRoundRequest) (results.OperationResult[*roundtypes.JoinRoundRequest, error], error) {
	return s.ValidateJoinRequest(ctx, req)
}
