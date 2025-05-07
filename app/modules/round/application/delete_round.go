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
		if payload.RoundID == sharedtypes.RoundID(uuid.Nil) { // Check if RoundID is zero
			return RoundOperationResult{
				Failure: &roundevents.RoundDeleteErrorPayload{
					RoundDeleteRequest: &payload,
					Error:              "round ID cannot be zero",
				},
			}, fmt.Errorf("round ID cannot be zero")
		}

		if payload.RequestingUserUserID == "" {
			return RoundOperationResult{
				Failure: &roundevents.RoundDeleteErrorPayload{
					RoundDeleteRequest: &payload,
					Error:              "requesting user's Discord ID cannot be empty",
				},
			}, fmt.Errorf("requesting user's Discord ID cannot be empty")
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
	// Add explicit nil UUID check
	if payload.RoundID == sharedtypes.RoundID(uuid.Nil) {
		s.logger.ErrorContext(ctx, "Cannot delete round with nil UUID")
		return RoundOperationResult{
			Failure: &roundevents.RoundDeleteErrorPayload{
				RoundDeleteRequest: &roundevents.RoundDeleteRequestPayload{
					RoundID: payload.RoundID,
				},
				Error: "round ID cannot be nil",
			},
		}, fmt.Errorf("cannot delete round: nil UUID provided")
	}

	s.logger.InfoContext(ctx, "DeleteRound service called",
		attr.RoundID("round_id", payload.RoundID),
	)

	// Log the actual UUID value for debugging
	s.logger.DebugContext(ctx, "Round ID details",
		attr.String("round_id_string", payload.RoundID.String()),
		attr.String("round_id_bytes", fmt.Sprintf("%v", payload.RoundID)),
	)

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
		}, fmt.Errorf("failed to delete round %s from DB: %w", payload.RoundID.String(), err)
	}
	s.logger.InfoContext(ctx, "Round deleted from DB", attr.RoundID("round_id", payload.RoundID))

	// Attempt to cancel any scheduled messages
	if err := s.EventBus.CancelScheduledMessage(ctx, payload.RoundID); err != nil {
		// Just log the error but don't fail the operation, as the round is already deleted
		s.logger.WarnContext(ctx, "Failed to cancel scheduled message",
			attr.RoundID("round_id", payload.RoundID),
			attr.Error(err),
		)
	} else {
		s.logger.InfoContext(ctx, "Scheduled message cancellation attempted", attr.RoundID("round_id", payload.RoundID))
	}

	s.logger.InfoContext(ctx, "Round deletion service process successful (Discord message ID expected in metadata)",
		attr.RoundID("round_id", payload.RoundID),
	)

	successPayload := &roundevents.RoundDeletedPayload{
		RoundID: payload.RoundID,
	}

	return RoundOperationResult{
		Success: successPayload,
	}, nil
}
