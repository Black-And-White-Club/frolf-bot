package roundservice

import (
	"context"
	"fmt"
	"strings"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
)

// ValidateRoundUpdateRequest validates the round update request.
func (s *RoundService) ValidateRoundUpdateRequest(ctx context.Context, payload roundevents.RoundUpdateRequestPayload) (RoundOperationResult, error) {
	return s.serviceWrapper(ctx, "ValidateRoundUpdateRequest", payload.RoundID, func(ctx context.Context) (RoundOperationResult, error) {
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
				Failure: &roundevents.RoundUpdateErrorPayload{
					RoundUpdateRequest: &payload,
					Error:              err.Error(),
				},
			}, nil // Return nil error since we're handling it in Failure
		}

		s.logger.InfoContext(ctx, "Round update request validated",
			attr.RoundID("round_id", payload.RoundID),
		)

		return RoundOperationResult{
			Success: &roundevents.RoundUpdateValidatedPayload{
				RoundUpdateRequestPayload: payload,
			},
		}, nil
	})
}

// UpdateRoundEntity updates the round entity with the new values.
func (s *RoundService) UpdateRoundEntity(ctx context.Context, payload roundevents.RoundUpdateValidatedPayload) (RoundOperationResult, error) {
	return s.serviceWrapper(ctx, "UpdateRoundEntity", payload.RoundUpdateRequestPayload.RoundID, func(ctx context.Context) (RoundOperationResult, error) {
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
			s.metrics.RecordDBOperationError(ctx, "GetRound")
			return RoundOperationResult{
				Failure: &roundevents.RoundUpdateErrorPayload{
					RoundUpdateRequest: &payload.RoundUpdateRequestPayload,
					Error:              err.Error(),
				},
			}, nil // Return nil error since we're handling it in Failure
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
			existingRound.StartTime = payload.RoundUpdateRequestPayload.StartTime
		}
		if payload.RoundUpdateRequestPayload.EventType != nil {
			existingRound.EventType = payload.RoundUpdateRequestPayload.EventType
		}

		// Update the round in the database
		if err := s.RoundDB.UpdateRound(ctx, existingRound.ID, existingRound); err != nil {
			s.logger.ErrorContext(ctx, "Failed to update round entity",
				attr.RoundID("round_id", payload.RoundUpdateRequestPayload.RoundID),
				attr.Error(err),
			)
			s.metrics.RecordDBOperationError(ctx, "UpdateRound")
			return RoundOperationResult{
				Failure: &roundevents.RoundUpdateErrorPayload{
					RoundUpdateRequest: &payload.RoundUpdateRequestPayload,
					Error:              err.Error(),
				},
			}, nil // Return nil error since we're handling it in Failure
		}

		s.logger.InfoContext(ctx, "Round entity updated successfully",
			attr.RoundID("round_id", payload.RoundUpdateRequestPayload.RoundID),
		)

		return RoundOperationResult{
			Success: &roundevents.RoundEntityUpdatedPayload{
				Round: *existingRound, // Dereference the existing round
			},
		}, nil
	})
}

// UpdateScheduledRoundEvents updates the scheduled events for a round.
func (s *RoundService) UpdateScheduledRoundEvents(ctx context.Context, payload roundevents.RoundScheduleUpdatePayload) (RoundOperationResult, error) {
	return s.serviceWrapper(ctx, "UpdateScheduledRoundEvents", payload.RoundID, func(ctx context.Context) (RoundOperationResult, error) {
		s.logger.InfoContext(ctx, "Processing scheduled round update",
			attr.RoundID("round_id", payload.RoundID),
		)

		// Step 1: Attempt to cancel existing scheduled events
		if err := s.EventBus.CancelScheduledMessage(ctx, payload.RoundID); err != nil {
			s.logger.WarnContext(ctx, "Failed to cancel existing scheduled events, proceeding anyway",
				attr.RoundID("round_id", payload.RoundID),
				attr.Error(err),
			)
		}

		// Step 2: Fetch the complete round information from the database
		s.logger.InfoContext(ctx, "DEBUG: About to fetch round from database",
			attr.RoundID("round_id", payload.RoundID),
		)

		round, fetchErr := s.RoundDB.GetRound(ctx, payload.RoundID)
		if fetchErr != nil {
			s.logger.ErrorContext(ctx, "Failed to fetch round for rescheduling",
				attr.RoundID("round_id", payload.RoundID),
				attr.Error(fetchErr),
			)
			s.metrics.RecordDBOperationError(ctx, "GetRound")

			// Create a new error payload for the failure
			return RoundOperationResult{
				Failure: &roundevents.RoundUpdateErrorPayload{
					RoundUpdateRequest: nil, // No valid request to return
					Error:              fetchErr.Error(),
				},
			}, nil // Return nil error since we're handling it in Failure
		}

		// Add debugging to see what we fetched
		if round == nil {
			s.logger.ErrorContext(ctx, "DEBUG: Round fetch returned nil pointer",
				attr.RoundID("round_id", payload.RoundID),
			)
			return RoundOperationResult{
				Failure: &roundevents.RoundUpdateErrorPayload{
					RoundUpdateRequest: nil,
					Error:              "round not found or is nil",
				},
			}, nil
		}

		s.logger.InfoContext(ctx, "DEBUG: Fetched round from database",
			attr.RoundID("round_id", round.ID),
			attr.String("title", string(round.Title)),
			attr.String("created_by", string(round.CreatedBy)),
			attr.Int("participants_count", len(round.Participants)),
		)

		// Validate the round has proper data
		if round.ID == (sharedtypes.RoundID{}) || round.ID.String() == "00000000-0000-0000-0000-000000000000" {
			s.logger.ErrorContext(ctx, "DEBUG: Round has zero-value ID",
				attr.RoundID("round_id", payload.RoundID),
				attr.RoundID("fetched_round_id", round.ID),
			)
			return RoundOperationResult{
				Failure: &roundevents.RoundUpdateErrorPayload{
					RoundUpdateRequest: nil,
					Error:              "round has invalid ID",
				},
			}, nil
		}

		s.logger.InfoContext(ctx, "Scheduled round update processed successfully",
			attr.RoundID("round_id", payload.RoundID),
		)

		// Step 3: Prepare the payload for the handler to publish
		// Since round is *roundtypes.Round, we need to dereference it
		roundValue := *round
		return RoundOperationResult{
			Success: &roundevents.RoundStoredPayload{
				Round: roundValue,
			},
		}, nil
	})
}
