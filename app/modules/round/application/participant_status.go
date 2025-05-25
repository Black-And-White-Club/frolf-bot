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
	result, err := s.serviceWrapper(ctx, "CheckParticipantStatus", payload.RoundID, func(ctx context.Context) (RoundOperationResult, error) {
		s.logger.InfoContext(ctx, "Checking participant status",
			attr.RoundID("round_id", payload.RoundID),
			attr.String("user_id", string(payload.UserID)),
			attr.String("requested_response", string(payload.Response)),
		)

		// Check if the user is already a participant
		participant, err := s.RoundDB.GetParticipant(ctx, payload.RoundID, payload.UserID)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to get participant's current status",
				attr.RoundID("round_id", payload.RoundID),
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
		s.logger.InfoContext(ctx, "Current participant status retrieved",
			attr.RoundID("round_id", payload.RoundID),
			attr.String("user_id", string(payload.UserID)),
			attr.String("current_status", currentStatus),
		)

		// Handle toggle removal (if the same status is clicked again)
		if currentStatus == string(payload.Response) {
			s.logger.InfoContext(ctx, "Toggle action detected - preparing removal request",
				attr.RoundID("round_id", payload.RoundID),
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
		s.logger.InfoContext(ctx, "Status change or new join detected - preparing validation request",
			attr.RoundID("round_id", payload.RoundID),
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

// ValidateParticipantJoinRequest validates the basic details of a join request and determines if the join is late.
func (s *RoundService) ValidateParticipantJoinRequest(ctx context.Context, payload roundevents.ParticipantJoinRequestPayload) (RoundOperationResult, error) {
	result, err := s.serviceWrapper(ctx, "ValidateParticipantJoinRequest", payload.RoundID, func(ctx context.Context) (RoundOperationResult, error) {
		s.logger.InfoContext(ctx, "Validating participant join request",
			attr.ExtractCorrelationID(ctx),
			attr.RoundID("round_id", payload.RoundID),
			attr.String("user_id", string(payload.UserID)),
			attr.String("response", string(payload.Response)),
			attr.Any("tag_number", payload.TagNumber),
			attr.Any("joined_late_payload", payload.JoinedLate),
		)

		var errorMessages []string
		// Basic validation checks
		if payload.RoundID == sharedtypes.RoundID(uuid.Nil) {
			errorMessages = append(errorMessages, "round ID cannot be nil")
		}
		if payload.UserID == "" {
			errorMessages = append(errorMessages, "participant Discord ID cannot be empty")
		}

		if len(errorMessages) > 0 {
			s.logger.ErrorContext(ctx, "Participant join request validation failed",
				attr.ExtractCorrelationID(ctx),
				attr.RoundID("round_id", payload.RoundID),
				attr.String("user_id", string(payload.UserID)),
				attr.Any("errors", errorMessages),
			)
			failurePayload := roundevents.RoundParticipantJoinErrorPayload{
				ParticipantJoinRequest: &payload,
				Error:                  fmt.Sprintf("validation failed: %v", errorMessages),
				EventMessageID:         "",
			}
			if payload.RoundID != sharedtypes.RoundID(uuid.Nil) {
				roundForError, getRoundErr := s.RoundDB.GetRound(ctx, payload.RoundID)
				if getRoundErr == nil {
					failurePayload.EventMessageID = roundForError.EventMessageID
				} else {
					s.logger.ErrorContext(ctx, "Failed to fetch round for EventMessageID in validation failure payload",
						attr.ExtractCorrelationID(ctx),
						attr.RoundID("round_id", payload.RoundID),
						attr.Error(getRoundErr),
					)
				}
			}
			return RoundOperationResult{Failure: failurePayload}, fmt.Errorf("validation failed: %v", errorMessages)
		}

		// Determine if Join is Late
		s.logger.InfoContext(ctx, "Fetching round details to determine if join is late",
			attr.ExtractCorrelationID(ctx),
			attr.RoundID("round_id", payload.RoundID),
		)
		round, err := s.RoundDB.GetRound(ctx, payload.RoundID)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to fetch round during join validation",
				attr.ExtractCorrelationID(ctx),
				attr.RoundID("round_id", payload.RoundID),
				attr.String("user_id", string(payload.UserID)),
				attr.Error(err),
			)
			failurePayload := roundevents.RoundParticipantJoinErrorPayload{
				ParticipantJoinRequest: &payload,
				Error:                  fmt.Sprintf("failed to fetch round details: %v", err),
				EventMessageID:         "",
			}
			if round != nil {
				failurePayload.EventMessageID = round.EventMessageID
			}

			return RoundOperationResult{Failure: failurePayload}, fmt.Errorf("failed to fetch round details: %w", err)
		}

		// Check if the state is 'InProgress' or 'Finalized'
		isLateJoin := round.State == roundtypes.RoundStateInProgress || round.State == roundtypes.RoundStateFinalized
		payload.JoinedLate = &isLateJoin // Update payload with determined status

		s.logger.InfoContext(ctx, "Determined late join status",
			attr.ExtractCorrelationID(ctx),
			attr.RoundID("round_id", payload.RoundID),
			attr.String("user_id", string(payload.UserID)),
			attr.String("round_state", string(round.State)),
			attr.Bool("is_late_join", isLateJoin),
		)

		switch payload.Response {
		case roundtypes.ResponseAccept, roundtypes.ResponseTentative:
			tagLookupPayload := roundevents.TagLookupRequestPayload{
				RoundID:    payload.RoundID,
				UserID:     payload.UserID,
				Response:   payload.Response,
				JoinedLate: payload.JoinedLate,
			}
			s.logger.InfoContext(ctx, "Validation successful for Accept/Tentative - Returning TagLookupPayload (pointer)", // Updated log message
				attr.ExtractCorrelationID(ctx),
				attr.Any("returning_payload", tagLookupPayload),
			)
			return RoundOperationResult{Success: &tagLookupPayload}, nil

		case roundtypes.ResponseDecline:

			s.logger.InfoContext(ctx, "Validation successful for Decline - Returning ParticipantJoinRequestPayload (pointer)",
				attr.ExtractCorrelationID(ctx),
				attr.Any("returning_payload", payload), // Log the validated payload
			)
			// Return a POINTER to the validated payload for the next handler
			return RoundOperationResult{Success: &payload}, nil // Return pointer

		default:
			// Handle unexpected response types
			s.logger.ErrorContext(ctx, "Unexpected response type in validated payload",
				attr.ExtractCorrelationID(ctx),
				attr.RoundID("round_id", payload.RoundID),
				attr.String("user_id", string(payload.UserID)),
				attr.String("response", string(payload.Response)),
			)
			failurePayload := roundevents.RoundParticipantJoinErrorPayload{
				ParticipantJoinRequest: &payload,
				Error:                  fmt.Sprintf("unexpected response type: %s", payload.Response),
				EventMessageID:         "",
			}
			if round != nil {
				failurePayload.EventMessageID = round.EventMessageID
			}

			return RoundOperationResult{Failure: failurePayload}, fmt.Errorf("unexpected response type: %s", payload.Response)
		}
	})

	return result, err
}

// ParticipantRemoval handles removing a participant from a round if they select the same RSVP response
func (s *RoundService) ParticipantRemoval(ctx context.Context, payload roundevents.ParticipantRemovalRequestPayload) (RoundOperationResult, error) {
	// The serviceWrapper handles the span, common metrics, and initial/final logging
	result, err := s.serviceWrapper(ctx, "ParticipantRemoval", payload.RoundID, func(ctx context.Context) (RoundOperationResult, error) {
		s.logger.InfoContext(ctx, "Processing participant removal",
			attr.ExtractCorrelationID(ctx), // Use ExtractCorrelationID from context
			attr.RoundID("round_id", payload.RoundID),
			attr.String("user_id", string(payload.UserID)),
		)

		// Retrieve round details to get EventMessageID (and potentially old status)
		// We need this before removal to get the EventMessageID for the payload.
		roundBeforeRemoval, err := s.RoundDB.GetRound(ctx, payload.RoundID)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to fetch round before participant removal from DB",
				attr.ExtractCorrelationID(ctx),
				attr.RoundID("round_id", payload.RoundID),
				attr.String("user_id", string(payload.UserID)),
				attr.Error(err),
			)
			failurePayload := roundevents.ParticipantRemovalErrorPayload{ // Assuming this payload exists
				RoundID: payload.RoundID,
				UserID:  payload.UserID,
				Error:   fmt.Sprintf("failed to fetch round details before removal: %v", err),
			}
			return RoundOperationResult{Failure: failurePayload}, fmt.Errorf("failed to fetch round details before removal: %w", err)
		}

		// Get current response before removing (optional, might be useful for context in logs/event if payload includes it)
		// You already have the round object, you can iterate its Participants slice directly
		var oldResponse string = "" // Default empty string
		var participantBeforeRemoval *roundtypes.Participant = nil
		for _, p := range roundBeforeRemoval.Participants {
			if p.UserID == payload.UserID {
				participantBeforeRemoval = &p
				oldResponse = string(p.Response)
				break
			}
		}

		if participantBeforeRemoval == nil {
			s.logger.WarnContext(ctx, "Attempted to remove participant who was not found in the round. No action needed.",
				attr.ExtractCorrelationID(ctx),
				attr.RoundID("round_id", payload.RoundID),
				attr.String("user_id", string(payload.UserID)),
			)
			// User wasn't in the round anyway. Return success with empty lists reflecting current state.
			// Ensure the EventMessageID is included even in this case.
			removedPayload := roundevents.ParticipantRemovedPayload{
				RoundID:               payload.RoundID,
				UserID:                payload.UserID,                    // Still include the user ID from the request
				EventMessageID:        roundBeforeRemoval.EventMessageID, // Use EventMessageID from fetched round, converted to string
				AcceptedParticipants:  []roundtypes.Participant{},        // Lists are empty
				DeclinedParticipants:  []roundtypes.Participant{},
				TentativeParticipants: []roundtypes.Participant{},
				// Include old Response if needed in the payload
				// Response: roundtypes.Response(oldResponse), // Will be "" if not found
			}
			s.logger.InfoContext(ctx, "--- Exiting ParticipantRemoval (Success: User Not Found) ---",
				attr.ExtractCorrelationID(ctx),
				attr.Any("returning_success_payload", removedPayload),
			)
			return RoundOperationResult{Success: removedPayload}, nil // Success, nothing was removed
		}

		s.logger.InfoContext(ctx, "Participant found before removal. Proceeding with removal.",
			attr.ExtractCorrelationID(ctx),
			attr.RoundID("round_id", payload.RoundID),
			attr.String("user_id", string(payload.UserID)),
			attr.String("old_response", oldResponse),
		)

		// Remove participant from database
		// Assuming RoundDB.RemoveParticipant takes RoundID and UserID and modifies the participants list in DB
		if err := s.RoundDB.RemoveParticipant(ctx, payload.RoundID, payload.UserID); err != nil {
			s.logger.ErrorContext(ctx, "Failed to remove participant from database",
				attr.ExtractCorrelationID(ctx),
				attr.RoundID("round_id", payload.RoundID),
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

		s.logger.InfoContext(ctx, "Participant removed successfully from database",
			attr.ExtractCorrelationID(ctx),
			attr.RoundID("round_id", payload.RoundID),
			attr.String("user_id", string(payload.UserID)),
		)

		// --- Get the UPDATED participant list AFTER removal ---
		// Assuming RoundDB.RemoveParticipant doesn't return the updated list, fetch the round again.
		// This fetch gets the current state of participants for the payload sent to Discord.
		roundAfterRemoval, err := s.RoundDB.GetRound(ctx, payload.RoundID)
		if err != nil {
			// This is a serious error - participant was removed, but we can't get the updated list to send to Discord.
			s.logger.ErrorContext(ctx, "Failed to fetch round AFTER participant removal to get updated lists",
				attr.ExtractCorrelationID(ctx),
				attr.RoundID("round_id", payload.RoundID),
				attr.String("user_id", string(payload.UserID)), // Log the user removed
				attr.Error(err),
			)
			// Decide how to handle this - ideally, the Discord embed should still update.
			// Maybe return success but with empty lists and an error flag? Or return a specific failure payload?
			// Returning a failure payload for the service operation seems appropriate.
			failurePayload := roundevents.ParticipantRemovalErrorPayload{
				RoundID: payload.RoundID,
				UserID:  payload.UserID,
				Error:   fmt.Sprintf("failed to fetch updated round after removal for discord update: %v", err),
			}
			// Returning this failure indicates the *entire* removal operation couldn't complete successfully
			// because the subsequent notification step failed.
			return RoundOperationResult{Failure: failurePayload}, fmt.Errorf("failed to fetch updated round after removal for discord update: %w", err)
		}

		// Categorize the participants from the UPDATED round's Participants slice
		accepted, declined, tentative := categorizeParticipants(roundAfterRemoval.Participants)

		// Prepare success payload including the updated lists and Discord message ID
		// Ensure this matches the UPDATED ParticipantRemovedPayload definition
		removedPayload := roundevents.ParticipantRemovedPayload{
			RoundID: payload.RoundID,
			UserID:  payload.UserID, // User who was removed
			// Include old Response if the payload definition has it and it's needed.
			// Response:       roundtypes.Response(oldResponse), // If the payload includes the Response field

			AcceptedParticipants:  accepted, // Include the updated lists
			DeclinedParticipants:  declined,
			TentativeParticipants: tentative,

			EventMessageID: roundAfterRemoval.EventMessageID,
			// Add JoinedLate if needed and available in the payload definition
			// JoinedLate: participantBeforeRemoval.JoinedLate, // If participant had this field
		}

		// --- Log the success payload being returned ---
		s.logger.InfoContext(ctx, "--- Exiting ParticipantRemoval (Success: User Removed) ---",
			attr.ExtractCorrelationID(ctx),
			attr.Any("returning_success_payload", removedPayload), // Log the specific removed payload with lists
			attr.Any("payload_round_id", removedPayload.RoundID),
			attr.Any("payload_accepted_count", len(removedPayload.AcceptedParticipants)),
			attr.Any("payload_declined_count", len(removedPayload.DeclinedParticipants)),
			attr.Any("payload_tentative_count", len(removedPayload.TentativeParticipants)),
		)
		// --- Return the success payload ---
		return RoundOperationResult{Success: removedPayload}, nil
	})

	// The serviceWrapper handles returning its result and error
	return result, err
}

// UpdateParticipantStatus handles participant status updates after validation
// Handles all response types (Accept, Decline, Tentative) with appropriate branching
func (s *RoundService) UpdateParticipantStatus(ctx context.Context, payload roundevents.ParticipantJoinRequestPayload) (RoundOperationResult, error) {
	// The serviceWrapper handles the span, common metrics, and initial/final logging
	result, err := s.serviceWrapper(ctx, "UpdateParticipantStatus", payload.RoundID, func(ctx context.Context) (RoundOperationResult, error) {
		// Safe handling of JoinedLate - provide a default value if nil
		isLateJoin := false
		if payload.JoinedLate != nil {
			isLateJoin = *payload.JoinedLate
		}

		// Initial processing log within the wrapped function
		s.logger.InfoContext(ctx, "Processing participant status update",
			attr.RoundID("round_id", payload.RoundID),
			attr.String("user_id", string(payload.UserID)),
			attr.String("response", string(payload.Response)),
			attr.Bool("is_late_join", isLateJoin),
			// Add tag number to log if present
			attr.Any("tag_number", payload.TagNumber), // Use attr.Any for pointer type
		)

		// --- Modified Logic Here ---
		// Scenario 1: TagNumber is provided AND response is Accept or Tentative
		// This path is likely triggered by a "tag found" event after a participant accepted/tentatived.
		if payload.TagNumber != nil && (payload.Response == roundtypes.ResponseAccept || payload.Response == roundtypes.ResponseTentative) {
			s.logger.InfoContext(ctx, "Handling participant update with PRE-EXISTING tag number",
				attr.RoundID("round_id", payload.RoundID),
				attr.String("user_id", string(payload.UserID)),
				attr.String("response", string(payload.Response)),
				attr.Int("tag_number", int(*payload.TagNumber)),
			)

			// Need round details for EventMessageID in the success/failure payload
			round, err := s.RoundDB.GetRound(ctx, payload.RoundID)
			if err != nil {
				s.logger.ErrorContext(ctx, "Failed to fetch round during participant update with tag",
					attr.RoundID("round_id", payload.RoundID),
					attr.String("user_id", string(payload.UserID)),
					attr.Error(err),
				)
				// Use a failure payload appropriate for an update error
				failurePayload := roundevents.ParticipantUpdateErrorPayload{
					RoundID: payload.RoundID,
					UserID:  payload.UserID,
					Error:   fmt.Sprintf("failed to fetch round details for tag update: %v", err),
				}
				return RoundOperationResult{Failure: failurePayload}, fmt.Errorf("failed to fetch round details for tag update: %w", err)
			}

			// Create participant struct for the database update
			participantToUpdate := roundtypes.Participant{
				UserID:    payload.UserID,
				Response:  payload.Response,
				TagNumber: payload.TagNumber, // Use the provided TagNumber
				Score:     nil,               // Assuming score isn't updated in this flow
			}

			// --- Call the DB layer to update the participant ---
			s.logger.InfoContext(ctx, "Calling DB.UpdateParticipant to update participant WITH tag",
				attr.RoundID("round_id", payload.RoundID),
				attr.String("user_id", string(payload.UserID)),
				attr.Int("tag_number", int(*payload.TagNumber)),
			)
			updatedParticipants, err := s.RoundDB.UpdateParticipant(ctx, payload.RoundID, participantToUpdate)
			if err != nil {
				s.logger.ErrorContext(ctx, "Failed to update participant with tag in DB",
					attr.RoundID("round_id", payload.RoundID),
					attr.String("user_id", string(payload.UserID)),
					attr.Error(err),
				)
				// Use the same failure payload structure as the Decline case for consistency
				failurePayload := roundevents.RoundParticipantJoinErrorPayload{ // This payload structure seems designed for join errors, adjust if needed for general updates
					ParticipantJoinRequest: &payload, // Might not fit perfectly if this isn't a join request flow
					Error:                  fmt.Sprintf("failed to update participant with tag in DB: %v", err),
					EventMessageID:         round.EventMessageID, // Use fetched message ID
				}
				return RoundOperationResult{Failure: failurePayload}, fmt.Errorf("failed to update participant with tag: %w", err)
			}

			// Categorize participants after the update
			accepted, declined, tentative := categorizeParticipants(updatedParticipants)

			// Prepare success payload (assuming ParticipantJoinedPayload works for this outcome)
			joinedPayload := roundevents.ParticipantJoinedPayload{
				RoundID:               payload.RoundID,
				AcceptedParticipants:  accepted,
				DeclinedParticipants:  declined,
				TentativeParticipants: tentative,
				EventMessageID:        round.EventMessageID, // Use fetched message ID
				JoinedLate:            &isLateJoin,          // Use the derived isLateJoin
			}
			return RoundOperationResult{Success: joinedPayload}, nil // Return success

		} else {
			// Scenario 2: No TagNumber provided OR response is Decline/Unknown.
			// Proceed with the existing switch logic.

			switch payload.Response {
			case roundtypes.ResponseAccept, roundtypes.ResponseTentative:
				// This case is now ONLY reached if TagNumber is nil and response is Accept/Tentative.
				// This flow triggers the tag lookup process.
				s.logger.InfoContext(ctx, "Handling participant Accept/Tentative (no tag provided), triggering tag lookup",
					attr.RoundID("round_id", payload.RoundID),
					attr.String("user_id", string(payload.UserID)),
					attr.String("response", string(payload.Response)),
				)
				// Return a payload indicating tag lookup is needed by another handler/process
				tagLookupPayload := roundevents.TagLookupRequestPayload{
					RoundID:    payload.RoundID,
					UserID:     payload.UserID,
					Response:   payload.Response,
					JoinedLate: payload.JoinedLate, // This is correct, as TagLookupRequestPayload can have nil
				}
				return RoundOperationResult{Success: tagLookupPayload}, nil

			case roundtypes.ResponseDecline:
				// Scenario 3: Participant Declined. Update participant directly.
				s.logger.InfoContext(ctx, "Handling participant Decline update",
					attr.RoundID("round_id", payload.RoundID),
					attr.String("user_id", string(payload.UserID)),
				)

				// Retrieve round details to get EventMessageID for result messages
				round, err := s.RoundDB.GetRound(ctx, payload.RoundID)
				if err != nil {
					s.logger.ErrorContext(ctx, "Failed to fetch round during participant decline update",
						attr.RoundID("round_id", payload.RoundID),
						attr.String("user_id", string(payload.UserID)),
						attr.Error(err),
					)
					failurePayload := roundevents.ParticipantUpdateErrorPayload{
						RoundID: payload.RoundID,
						UserID:  payload.UserID,
						Error:   fmt.Sprintf("failed to fetch round details for decline update: %v", err),
					}
					return RoundOperationResult{Failure: failurePayload}, fmt.Errorf("failed to fetch round details for decline update: %w", err)
				}

				// Create participant struct for database update - TagNumber is nil for Decline
				participantToUpdate := roundtypes.Participant{
					UserID:    payload.UserID,
					Response:  payload.Response,
					TagNumber: nil, // No tag needed for Decline
					Score:     nil, // Assuming score isn't updated here
					// Other fields like Name, IsBot etc.
				}

				// --- Call the DB layer to update the participant ---
				s.logger.InfoContext(ctx, "Calling DB.UpdateParticipant to update participant with DECLINE status",
					attr.RoundID("round_id", payload.RoundID),
					attr.String("user_id", string(payload.UserID)),
				)
				updatedParticipants, err := s.RoundDB.UpdateParticipant(ctx, payload.RoundID, participantToUpdate)
				if err != nil {
					s.logger.ErrorContext(ctx, "Failed to update participant with Decline in DB",
						attr.RoundID("round_id", payload.RoundID),
						attr.String("user_id", string(payload.UserID)),
						attr.Error(err),
					)
					// Use existing error payload structure
					failurePayload := roundevents.RoundParticipantJoinErrorPayload{
						ParticipantJoinRequest: &payload,
						Error:                  fmt.Sprintf("failed to update participant status in DB: %v", err),
						EventMessageID:         round.EventMessageID,
					}
					return RoundOperationResult{Failure: failurePayload}, fmt.Errorf("failed to update participant status: %w", err)
				}

				// Categorize participants after the update
				accepted, declined, tentative := categorizeParticipants(updatedParticipants)

				// Prepare success payload (reuse ParticipantJoinedPayload)
				joinedPayload := roundevents.ParticipantJoinedPayload{
					RoundID:               payload.RoundID,
					AcceptedParticipants:  accepted,
					DeclinedParticipants:  declined,
					TentativeParticipants: tentative,
					EventMessageID:        round.EventMessageID,
					JoinedLate:            &isLateJoin, // Use the derived isLateJoin
				}
				return RoundOperationResult{Success: joinedPayload}, nil // Return success

			default:
				// Scenario 4: Unknown response type
				s.logger.ErrorContext(ctx, "Unknown response type in payload",
					attr.RoundID("round_id", payload.RoundID),
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
		}
	})

	// The serviceWrapper handles returning its result and error
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
