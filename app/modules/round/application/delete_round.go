package roundservice

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
)

// ValidateRoundDeleteRequest validates the round delete request.
func (s *RoundService) ValidateRoundDeleteRequest(ctx context.Context, payload roundevents.RoundDeleteRequestPayload) (RoundOperationResult, error) {
	return s.serviceWrapper(ctx, "ValidateRoundDeleteRequest", payload.RoundID, func(ctx context.Context) (RoundOperationResult, error) {
		if payload.RoundID == sharedtypes.RoundID(uuid.Nil) {
			return RoundOperationResult{
				Failure: &roundevents.RoundDeleteErrorPayload{
					RoundDeleteRequest: &payload,
					Error:              "round ID cannot be zero",
				},
			}, nil
		}

		if payload.RequestingUserUserID == "" {
			return RoundOperationResult{
				Failure: &roundevents.RoundDeleteErrorPayload{
					RoundDeleteRequest: &payload,
					Error:              "requesting user's Discord ID cannot be empty",
				},
			}, nil
		}

		round, err := s.RoundDB.GetRound(ctx, payload.RoundID)
		if err != nil {
			s.logger.WarnContext(ctx, "Round not found for delete request",
				attr.String("round_id", payload.RoundID.String()),
				attr.String("requesting_user", string(payload.RequestingUserUserID)),
				attr.Error(err),
			)
			return RoundOperationResult{
				Failure: &roundevents.RoundDeleteErrorPayload{
					RoundDeleteRequest: &payload,
					Error:              fmt.Sprintf("round not found: %v", err),
				},
			}, nil
		}

		if round.CreatedBy != payload.RequestingUserUserID {
			s.logger.WarnContext(ctx, "Unauthorized delete request",
				attr.String("round_id", payload.RoundID.String()),
				attr.String("requesting_user", string(payload.RequestingUserUserID)),
				attr.String("round_created_by", string(round.CreatedBy)),
			)
			return RoundOperationResult{
				Failure: &roundevents.RoundDeleteErrorPayload{
					RoundDeleteRequest: &payload,
					Error:              "unauthorized: only the round creator can delete the round",
				},
			}, nil
		}

		s.logger.InfoContext(ctx, "Round delete request validated",
			attr.String("round_id", payload.RoundID.String()),
			attr.String("requesting_user", string(payload.RequestingUserUserID)),
		)

		return RoundOperationResult{
			Success: &roundevents.RoundDeleteValidatedPayload{
				RoundDeleteRequestPayload: payload,
			},
		}, nil
	})
}

func (s *RoundService) DeleteRound(ctx context.Context, payload roundevents.RoundDeleteAuthorizedPayload) (RoundOperationResult, error) {
	if payload.RoundID == sharedtypes.RoundID(uuid.Nil) {
		s.logger.ErrorContext(ctx, "Cannot delete round with nil UUID")
		return RoundOperationResult{
			Failure: &roundevents.RoundDeleteErrorPayload{
				RoundDeleteRequest: &roundevents.RoundDeleteRequestPayload{
					RoundID: payload.RoundID,
				},
				Error: "round ID cannot be nil",
			},
		}, nil
	}

	s.logger.InfoContext(ctx, "DeleteRound service called",
		attr.RoundID("round_id", payload.RoundID),
	)

	round, err := s.RoundDB.GetRound(ctx, payload.RoundID)
	if err != nil {
		s.logger.WarnContext(ctx, "Cannot delete non-existent round",
			attr.RoundID("round_id", payload.RoundID),
			attr.Error(err),
		)
		return RoundOperationResult{
			Failure: &roundevents.RoundDeleteErrorPayload{
				RoundDeleteRequest: &roundevents.RoundDeleteRequestPayload{
					RoundID: payload.RoundID,
				},
				Error: fmt.Sprintf("round not found: %v", err),
			},
		}, nil
	}

	// Capture the event message ID before deletion
	eventMessageID := round.EventMessageID

	// Delete the round from the database
	if err := s.RoundDB.DeleteRound(ctx, payload.RoundID); err != nil {
		s.logger.ErrorContext(ctx, "Failed to delete round from DB",
			attr.RoundID("round_id", payload.RoundID),
			attr.Error(err),
		)
		return RoundOperationResult{
			Failure: &roundevents.RoundDeleteErrorPayload{
				RoundDeleteRequest: &roundevents.RoundDeleteRequestPayload{
					RoundID: payload.RoundID,
				},
				Error: fmt.Sprintf("failed to delete round from database: %v", err),
			},
		}, nil
	}

	s.logger.InfoContext(ctx, "Round deleted from DB", attr.RoundID("round_id", payload.RoundID))

	// Attempt to cancel any scheduled jobs for this round
	if err := s.QueueService.CancelRoundJobs(ctx, payload.RoundID); err != nil {
		s.logger.WarnContext(ctx, "Failed to cancel scheduled jobs",
			attr.RoundID("round_id", payload.RoundID),
			attr.Error(err),
		)
	} else {
		s.logger.InfoContext(ctx, "Scheduled jobs cancellation successful", attr.RoundID("round_id", payload.RoundID))
	}

	s.logger.InfoContext(ctx, "Round deletion service process successful",
		attr.RoundID("round_id", payload.RoundID),
	)

	successPayload := &roundevents.RoundDeletedPayload{
		RoundID:        payload.RoundID,
		EventMessageID: eventMessageID,
	}

	return RoundOperationResult{
		Success: successPayload,
	}, nil
}
