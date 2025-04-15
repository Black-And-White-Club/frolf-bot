package roundservice

import (
	"context"
	"errors"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
)

// ValidateRoundDeleteRequest validates the round delete request.
func (s *RoundService) ValidateRoundDeleteRequest(ctx context.Context, payload roundevents.RoundDeleteRequestPayload) (RoundOperationResult, error) {
	return s.serviceWrapper(ctx, "ValidateRoundDeleteRequest", func() (RoundOperationResult, error) {
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
	// Get the event message ID
	eventMessageID, err := s.RoundDB.GetEventMessageID(ctx, payload.RoundID)
	if err != nil {
		return RoundOperationResult{
			Failure: &roundevents.RoundDeleteErrorPayload{
				RoundDeleteRequest: &roundevents.RoundDeleteRequestPayload{
					RoundID:              payload.RoundID,
					RequestingUserUserID: "",
				},
				Error: "failed to retrieve EventMessageID for round",
			},
		}, errors.New("failed to retrieve EventMessageID for round")
	}

	// Delete the round
	if err := s.RoundDB.DeleteRound(ctx, payload.RoundID); err != nil {
		s.logger.ErrorContext(ctx, "Failed to delete round %s: %v", attr.RoundID("round_id", payload.RoundID), attr.Error(err))
		return RoundOperationResult{
			Failure: &roundevents.RoundDeleteErrorPayload{
				RoundDeleteRequest: &roundevents.RoundDeleteRequestPayload{
					RoundID:              payload.RoundID,
					RequestingUserUserID: "",
				},
				Error: "failed to delete round",
			},
		}, errors.New("failed to delete round")
	}

	// Cancel the scheduled message
	if err := s.EventBus.CancelScheduledMessage(ctx, payload.RoundID); err != nil {
		s.logger.ErrorContext(ctx, "Failed to cancel scheduled message for round %s: %v", attr.RoundID("round_id", payload.RoundID), attr.Error(err))
		return RoundOperationResult{
			Failure: &roundevents.RoundDeleteErrorPayload{
				RoundDeleteRequest: &roundevents.RoundDeleteRequestPayload{
					RoundID:              payload.RoundID,
					RequestingUserUserID: "",
				},
				Error: "failed to cancel scheduled messages",
			},
		}, errors.New("failed to cancel scheduled messages")
	}

	return RoundOperationResult{
		Success: &roundevents.RoundDeletedPayload{
			RoundID:        payload.RoundID,
			EventMessageID: *eventMessageID,
		},
	}, nil
}
