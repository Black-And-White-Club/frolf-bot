package roundservice

import (
	"context"
	"fmt"
	"strings"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
)

// ValidateRoundUpdateRequest validates the round update request.
func (s *RoundService) ValidateRoundUpdateRequest(ctx context.Context, payload roundevents.RoundUpdateRequestPayload) (RoundOperationResult, error) {
	return s.serviceWrapper(ctx, "ValidateRoundUpdateRequest", func() (RoundOperationResult, error) {
		var errs []string
		if payload.RoundID == sharedtypes.RoundID(uuid.Nil) {
			errs = append(errs, "round ID cannot be zero")
		}

		if payload.Title == "" && payload.Description == nil && payload.Location == nil && payload.StartTime == nil {
			errs = append(errs, "at least one field to update must be provided")
		}

		if len(errs) > 0 {
			err := fmt.Errorf("validation errors: %s", strings.Join(errs, "; "))
			s.logger.ErrorContext(ctx, "Round update request validation failed",
				attr.RoundID("round_id", payload.RoundID),
				attr.Error(err),
			)
			return RoundOperationResult{
				Failure: roundevents.RoundUpdateErrorPayload{
					RoundUpdateRequest: &payload,
					Error:              err.Error(),
				},
			}, err
		}

		s.logger.InfoContext(ctx, "Round update request validated",
			attr.RoundID("round_id", payload.RoundID),
		)

		return RoundOperationResult{
			Success: roundevents.RoundUpdateValidatedPayload{
				RoundUpdateRequestPayload: payload,
			},
		}, nil
	})
}

// UpdateRoundEntity updates the round entity with the new values.
func (s *RoundService) UpdateRoundEntity(ctx context.Context, payload roundevents.RoundUpdateValidatedPayload) (RoundOperationResult, error) {
	return s.serviceWrapper(ctx, "UpdateRoundEntity", func() (RoundOperationResult, error) {
		s.logger.InfoContext(ctx, "Updating round entity",
			attr.RoundID("round_id", payload.RoundUpdateRequestPayload.RoundID),
		)

		// Fetch the existing round
		existingRound, err := s.RoundDB.GetRound(ctx, payload.RoundUpdateRequestPayload.RoundID)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to fetch round",
				attr.RoundID("round_id", payload.RoundUpdateRequestPayload.RoundID),
				attr.Error(err),
			)
			return RoundOperationResult{
				Failure: roundevents.RoundUpdateErrorPayload{
					RoundUpdateRequest: &payload.RoundUpdateRequestPayload,
					Error:              err.Error(),
				},
			}, err
		}

		// Apply updates
		if payload.RoundUpdateRequestPayload.Title != "" {
			existingRound.Title = payload.RoundUpdateRequestPayload.Title
		}
		if payload.RoundUpdateRequestPayload.Description != nil {
			existingRound.Description = payload.RoundUpdateRequestPayload.Description
		}
		if payload.RoundUpdateRequestPayload.Location != nil {
			existingRound.Location = payload.RoundUpdateRequestPayload.Location
		}
		if payload.RoundUpdateRequestPayload.StartTime != nil {
			startTime := sharedtypes.StartTime(*payload.RoundUpdateRequestPayload.StartTime)
			existingRound.StartTime = &startTime
		}

		// Update the round in the database
		if err := s.RoundDB.UpdateRound(ctx, existingRound.ID, existingRound); err != nil {
			s.logger.ErrorContext(ctx, "Failed to update round entity",
				attr.RoundID("round_id", payload.RoundUpdateRequestPayload.RoundID),
				attr.Error(err),
			)
			return RoundOperationResult{
				Failure: roundevents.RoundUpdateErrorPayload{
					RoundUpdateRequest: &payload.RoundUpdateRequestPayload,
					Error:              err.Error(),
				},
			}, err
		}

		s.logger.InfoContext(ctx, "Round entity updated successfully",
			attr.RoundID("round_id", payload.RoundUpdateRequestPayload.RoundID),
		)

		return RoundOperationResult{
			Success: roundevents.RoundEntityUpdatedPayload{
				Round: roundtypes.Round{
					ID:           existingRound.ID,
					Title:        existingRound.Title,
					Description:  existingRound.Description,
					Location:     existingRound.Location,
					EventType:    existingRound.EventType,
					StartTime:    existingRound.StartTime,
					Finalized:    existingRound.Finalized,
					CreatedBy:    existingRound.CreatedBy,
					State:        existingRound.State,
					Participants: existingRound.Participants,
				},
			},
		}, nil
	})
}

// UpdateScheduledRoundEvents updates the scheduled events for a round.
func (s *RoundService) UpdateScheduledRoundEvents(ctx context.Context, payload roundevents.RoundScheduleUpdatePayload) (RoundOperationResult, error) {
	return s.serviceWrapper(ctx, "UpdateScheduledRoundEvents", func() (RoundOperationResult, error) {
		s.logger.InfoContext(ctx, "Processing scheduled round update",
			attr.RoundID("round_id", payload.RoundID),
		)

		// Step 1: Attempt to cancel existing scheduled events
		if err := s.EventBus.CancelScheduledMessage(ctx, payload.RoundID); err != nil {
			s.logger.Warn("Failed to cancel existing scheduled events, proceeding anyway",
				attr.RoundID("round_id", payload.RoundID),
				attr.Error(err),
			)
		}

		// Step 2: Fetch the complete round information from the database
		round, fetchErr := s.RoundDB.GetRound(ctx, payload.RoundID)
		if fetchErr != nil {
			s.logger.ErrorContext(ctx, "Failed to fetch round for rescheduling",
				attr.RoundID("round_id", payload.RoundID),
				attr.Error(fetchErr),
			)

			// Create a new error payload for the failure
			errorPayload := roundevents.RoundUpdateErrorPayload{
				RoundUpdateRequest: nil, // No valid request to return
				Error:              fetchErr.Error(),
			}

			return RoundOperationResult{
				Failure: errorPayload,
			}, fetchErr
		}

		// Step 3: Prepare the payload for the handler to publish
		storedPayload := roundevents.RoundStoredPayload{
			Round: *round,
		}

		// Return the payload for the handler to use for rescheduling
		return RoundOperationResult{
			Success: storedPayload,
		}, nil
	})
}
