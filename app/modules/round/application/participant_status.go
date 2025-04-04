package roundservice

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
)

// CheckParticipantStatus checks if a join request is a toggle or requires validation.
func (s *RoundService) CheckParticipantStatus(ctx context.Context, payload roundevents.ParticipantJoinRequestPayload) (RoundOperationResult, error) {
	result, err := s.serviceWrapper(ctx, "CheckParticipantStatus", func() (RoundOperationResult, error) {
		s.logger.Info("Checking participant status",
			attr.StringUUID("round_id", payload.RoundID.String()),
			attr.String("user_id", string(payload.UserID)),
			attr.String("requested_response", string(payload.Response)),
		)

		// Check if the user is already a participant
		participant, err := s.RoundDB.GetParticipant(ctx, payload.RoundID, payload.UserID)
		if err != nil {
			s.logger.Error("Failed to get participant's current status",
				attr.StringUUID("round_id", payload.RoundID.String()),
				attr.String("user_id", string(payload.UserID)),
				attr.Error(err),
			)
			// Use a specific error payload
			failurePayload := roundevents.ParticipantStatusCheckErrorPayload{
				RoundID: payload.RoundID,
				UserID:  payload.UserID,
				Error:   fmt.Sprintf("failed to get participant status: %v", err),
			}
			return RoundOperationResult{Failure: failurePayload}, fmt.Errorf("failed to get participant status: %w", err)
		}

		currentStatus := ""
		if participant != nil {
			currentStatus = string(participant.Response)
		}
		s.logger.Info("Current participant status retrieved",
			attr.StringUUID("round_id", payload.RoundID.String()),
			attr.String("user_id", string(payload.UserID)),
			attr.String("current_status", currentStatus),
		)

		// Handle toggle removal (if the same status is clicked again)
		if currentStatus == string(payload.Response) {
			s.logger.Info("Toggle action detected - preparing removal request",
				attr.StringUUID("round_id", payload.RoundID.String()),
				attr.String("user_id", string(payload.UserID)),
				attr.String("status", currentStatus),
			)
			// Prepare a removal request payload for the caller
			removalPayload := roundevents.ParticipantRemovalRequestPayload{
				RoundID: payload.RoundID,
				UserID:  payload.UserID,
			}
			return RoundOperationResult{Success: removalPayload}, nil
		}

		// Otherwise, prepare a validation request payload
		s.logger.Info("Status change or new join detected - preparing validation request",
			attr.StringUUID("round_id", payload.RoundID.String()),
			attr.String("user_id", string(payload.UserID)),
		)
		// Pass the original payload forward, wrapped in a type indicating validation is next
		validationPayload := roundevents.ParticipantJoinValidationRequestPayload{
			RoundID:  payload.RoundID,
			UserID:   payload.UserID,
			Response: payload.Response,
		}
		return RoundOperationResult{Success: validationPayload}, nil
	})

	return result, err
}

// ValidateParticipantJoinRequest validates the basic details and determines if the join is late.
func (s *RoundService) ValidateParticipantJoinRequest(ctx context.Context, payload roundevents.ParticipantJoinRequestPayload) (RoundOperationResult, error) {
	result, err := s.serviceWrapper(ctx, "ValidateParticipantJoinRequest", func() (RoundOperationResult, error) {
		s.logger.Info("Validating participant join request",
			attr.StringUUID("round_id", payload.RoundID.String()),
			attr.String("user_id", string(payload.UserID)),
			attr.String("response", string(payload.Response)),
		)

		var errorMessages []string
		// Basic validation checks
		if payload.RoundID == sharedtypes.RoundID(uuid.Nil) {
			errorMessages = append(errorMessages, "round ID cannot be nil")
		}
		if payload.UserID == "" {
			errorMessages = append(errorMessages, "participant Discord ID cannot be empty")
		}
		// Add any other necessary validations

		if len(errorMessages) > 0 {
			s.logger.Error("Participant join request validation failed",
				attr.StringUUID("round_id", payload.RoundID.String()),
				attr.String("user_id", string(payload.UserID)),
				attr.Any("errors", errorMessages),
			)
			failurePayload := roundevents.RoundParticipantJoinErrorPayload{
				ParticipantJoinRequest: &payload,
				Error:                  fmt.Sprintf("validation failed: %v", errorMessages),
				EventMessageID:         sharedtypes.RoundID(uuid.Nil),
			}
			return RoundOperationResult{Failure: failurePayload}, fmt.Errorf("validation failed: %v", errorMessages)
		}

		// --- Determine if Join is Late ---
		s.logger.Info("Fetching round details to determine if join is late",
			attr.StringUUID("round_id", payload.RoundID.String()),
		)
		round, err := s.RoundDB.GetRound(ctx, payload.RoundID)
		if err != nil {
			// Handle error fetching round
			s.logger.Error("Failed to fetch round during join validation",
				attr.StringUUID("round_id", payload.RoundID.String()),
				attr.String("user_id", string(payload.UserID)),
				attr.Error(err),
			)
			failurePayload := roundevents.RoundParticipantJoinErrorPayload{
				ParticipantJoinRequest: &payload,
				Error:                  fmt.Sprintf("failed to fetch round details: %v", err),
			}
			return RoundOperationResult{Failure: failurePayload}, fmt.Errorf("failed to fetch round details: %w", err)
		}

		// Example: Check if the state is 'InProgress' or 'Finalized'
		isLateJoin := round.State == roundtypes.RoundStateInProgress || round.State == roundtypes.RoundStateFinalized

		s.logger.Info("Determined late join status",
			attr.StringUUID("round_id", payload.RoundID.String()),
			attr.String("user_id", string(payload.UserID)),
			attr.String("round_state", string(round.State)),
			attr.Bool("is_late_join", isLateJoin),
		)

		// Create validated request with late join status
		validatedRequest := roundevents.ParticipantJoinRequestPayload{
			RoundID:    payload.RoundID,
			UserID:     payload.UserID,
			Response:   payload.Response,
			JoinedLate: &isLateJoin,
		}

		// Determine appropriate next step based on response type
		switch payload.Response {
		case roundtypes.ResponseAccept, roundtypes.ResponseTentative:
			// Need tag lookup for Accept/Tent ative responses
			return RoundOperationResult{Success: validatedRequest}, nil
		case roundtypes.ResponseDecline:
			// Directly proceed to update participant status
			return RoundOperationResult{Success: validatedRequest}, nil
		default:
			// Handle unexpected response types
			failurePayload := roundevents.RoundParticipantJoinErrorPayload{
				ParticipantJoinRequest: &payload,
				Error:                  fmt.Sprintf("unexpected response type: %s", payload.Response),
			}
			return RoundOperationResult{Failure: failurePayload}, fmt.Errorf("unexpected response type: %s", payload.Response)
		}
	})

	return result, err
}

// ParticipantRemoval handles removing a participant from a round if they select the same RSVP response
func (s *RoundService) ParticipantRemoval(ctx context.Context, payload roundevents.ParticipantRemovalRequestPayload) (RoundOperationResult, error) {
	result, err := s.serviceWrapper(ctx, "ParticipantRemoval", func() (RoundOperationResult, error) {
		s.logger.Info("Attempting to remove participant",
			attr.StringUUID("round_id", payload.RoundID.String()),
			attr.String("user_id", string(payload.UserID)),
		)

		// Retrieve round details to get EventMessageID
		round, err := s.RoundDB.GetRound(ctx, payload.RoundID)
		if err != nil {
			s.logger.Error("Failed to fetch round during participant removal",
				attr.StringUUID("round_id", payload.RoundID.String()),
				attr.String("user_id", string(payload.UserID)),
				attr.Error(err),
			)
			failurePayload := roundevents.ParticipantRemovalErrorPayload{
				RoundID: payload.RoundID,
				UserID:  payload.UserID,
				Error:   fmt.Sprintf("failed to fetch round details: %v", err),
			}
			return RoundOperationResult{Failure: failurePayload}, fmt.Errorf("failed to fetch round details: %w", err)
		}

		// Get current response before removing (for event payload)
		participant, err := s.RoundDB.GetParticipant(ctx, payload.RoundID, payload.UserID)
		if err != nil {
			s.logger.Error("Failed to get participant response before removal",
				attr.StringUUID("round_id", payload.RoundID.String()),
				attr.String("user_id", string(payload.UserID)),
				attr.Error(err),
			)
			failurePayload := roundevents.ParticipantRemovalErrorPayload{
				RoundID: payload.RoundID,
				UserID:  payload.UserID,
				Error:   fmt.Sprintf("failed to get participant response before removal: %v", err),
			}
			return RoundOperationResult{Failure: failurePayload}, fmt.Errorf("failed to get participant response before removal: %w", err)
		}

		// Determine the response to include in the event
		var response roundtypes.Response
		if participant == nil {
			response = roundtypes.Response("") // Indicate user wasn't found
			s.logger.Warn("Attempted to remove participant who was not found in the round",
				attr.StringUUID("round_id", payload.RoundID.String()),
				attr.String("user_id", string(payload.UserID)),
			)
		} else {
			response = roundtypes.Response(participant.Response)
		}

		// Remove participant from database
		if err := s.RoundDB.RemoveParticipant(ctx, payload.RoundID, payload.UserID); err != nil {
			s.logger.Error("Failed to remove participant from database",
				attr.StringUUID("round_id", payload.RoundID.String()),
				attr.String("user_id", string(payload.UserID)),
				attr.Error(err),
			)
			failurePayload := roundevents.ParticipantRemovalErrorPayload{
				RoundID: payload.RoundID,
				UserID:  payload.UserID,
				Error:   fmt.Sprintf("failed to remove participant from database: %v", err),
			}
			return RoundOperationResult{Failure: failurePayload}, fmt.Errorf("failed to remove participant from database: %w", err)
		}

		s.logger.Info("Participant removed successfully from database",
			attr.StringUUID("round_id", payload.RoundID.String()),
			attr.String("user_id", string(payload.UserID)),
		)
		// Prepare success payload
		removedPayload := roundevents.ParticipantRemovedPayload{
			RoundID:        payload.RoundID,
			UserID:         payload.UserID,
			Response:       response, // Include the status they *had* before removal
			EventMessageID: round.EventMessageID,
		}
		return RoundOperationResult{Success: removedPayload}, nil
	})

	return result, err
}

// UpdateParticipantStatus handles participant status updates after validation
// Handles all response types (Accept, Decline, Tentative) with appropriate branching
func (s *RoundService) UpdateParticipantStatus(ctx context.Context, payload roundevents.ParticipantJoinRequestPayload) (RoundOperationResult, error) {
	result, err := s.serviceWrapper(ctx, "UpdateParticipantStatus", func() (RoundOperationResult, error) {
		// Safe handling of JoinedLate - provide a default value if nil
		isLateJoin := false
		if payload.JoinedLate != nil {
			isLateJoin = *payload.JoinedLate
		}

		s.logger.Info("Processing participant status update",
			attr.StringUUID("round_id", payload.RoundID.String()),
			attr.String("user_id", string(payload.UserID)),
			attr.String("response", string(payload.Response)),
			attr.Bool("is_late_join", isLateJoin),
		)

		// Branch based on response type
		switch payload.Response {
		case roundtypes.ResponseAccept, roundtypes.ResponseTentative:
			// For Accept/Tentative, we should trigger tag lookup
			// Return a payload indicating tag lookup is needed
			tagLookupPayload := roundevents.TagLookupRequestPayload{
				RoundID:    payload.RoundID,
				UserID:     payload.UserID,
				Response:   payload.Response,
				JoinedLate: payload.JoinedLate,
			}
			return RoundOperationResult{Success: tagLookupPayload}, nil

		case roundtypes.ResponseDecline:
			// For Decline, we directly update the participant without tag lookup
			// Retrieve round details to get EventMessageID
			round, err := s.RoundDB.GetRound(ctx, payload.RoundID)
			if err != nil {
				s.logger.Error("Failed to fetch round during participant update",
					attr.StringUUID("round_id", payload.RoundID.String()),
					attr.String("user_id", string(payload.UserID)),
					attr.Error(err),
				)
				failurePayload := roundevents.ParticipantUpdateErrorPayload{
					RoundID: payload.RoundID,
					UserID:  payload.UserID,
					Error:   fmt.Sprintf("failed to fetch round details: %v", err),
				}
				return RoundOperationResult{Failure: failurePayload}, fmt.Errorf("failed to fetch round details: %w", err)
			}

			// Create participant struct for update
			participant := roundtypes.Participant{
				UserID:    payload.UserID,
				Response:  payload.Response,
				TagNumber: nil, // No tag needed for Decline
				Score:     nil,
			}

			// Update participant in the database
			updatedParticipants, err := s.RoundDB.UpdateParticipant(ctx, payload.RoundID, participant)
			if err != nil {
				s.logger.Error("Failed to update participant status for decline",
					attr.StringUUID("round_id", payload.RoundID.String()),
					attr.String("user_id", string(payload.UserID)),
					attr.Error(err),
				)
				failurePayload := roundevents.RoundParticipantJoinErrorPayload{
					ParticipantJoinRequest: &payload,
					Error:                  fmt.Sprintf("failed to update participant status in DB: %v", err),
					EventMessageID:         round.EventMessageID,
				}
				return RoundOperationResult{Failure: failurePayload}, fmt.Errorf("failed to update participant status: %w", err)
			}

			// Categorize participants
			accepted, declined, tentative := categorizeParticipants(updatedParticipants)

			// Prepare success payload with all participant lists
			joinedPayload := roundevents.ParticipantJoinedPayload{
				RoundID:               payload.RoundID,
				AcceptedParticipants:  accepted,
				DeclinedParticipants:  declined,
				TentativeParticipants: tentative,
				EventMessageID:        round.EventMessageID,
				JoinedLate:            payload.JoinedLate,
			}
			return RoundOperationResult{Success: joinedPayload}, nil

		default:
			// Handle unknown response types
			s.logger.Error("Unknown response type",
				attr.StringUUID("round_id", payload.RoundID.String()),
				attr.String("user_id", string(payload.UserID)),
				attr.String("response", string(payload.Response)),
			)
			failurePayload := roundevents.ParticipantUpdateErrorPayload{
				RoundID: payload.RoundID,
				UserID:  payload.UserID,
				Error:   fmt.Sprintf("unknown response type: %s", payload.Response),
			}
			return RoundOperationResult{Failure: failurePayload}, fmt.Errorf("unknown response type: %s", payload.Response)
		}
	})

	return result, err
}

// categorizeParticipants is a helper to split participants by response status.
func categorizeParticipants(participants []roundtypes.Participant) (accepted, declined, tentative []roundtypes.Participant) {
	for _, p := range participants {
		switch roundtypes.Response(p.Response) {
		case roundtypes.ResponseAccept:
			accepted = append(accepted, p)
		case roundtypes.ResponseDecline:
			declined = append(declined, p)
		case roundtypes.ResponseTentative:
			tentative = append(tentative, p)
		}
	}
	return
}
