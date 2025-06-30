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
		participant, err := s.RoundDB.GetParticipant(ctx, payload.GuildID, payload.RoundID, payload.UserID)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to get participant's current status",
				attr.RoundID("round_id", payload.RoundID),
				attr.String("user_id", string(payload.UserID)),
				attr.Error(err),
			)
			// Use a specific error payload
			failurePayload := &roundevents.ParticipantStatusCheckErrorPayload{
				RoundID: payload.RoundID,
				UserID:  payload.UserID,
				Error:   fmt.Sprintf("failed to get participant status: %v", err),
			}
			return RoundOperationResult{Failure: failurePayload}, nil
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
			removalPayload := &roundevents.ParticipantRemovalRequestPayload{
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
		validationPayload := &roundevents.ParticipantJoinValidationRequestPayload{
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
			s.logger.InfoContext(ctx, "Participant join request validation failed - returning failure payload",
				attr.ExtractCorrelationID(ctx),
				attr.RoundID("round_id", payload.RoundID),
				attr.String("user_id", string(payload.UserID)),
				attr.Any("errors", errorMessages),
			)
			failurePayload := &roundevents.RoundParticipantJoinErrorPayload{
				ParticipantJoinRequest: &payload,
				Error:                  fmt.Sprintf("validation failed: %v", errorMessages),
				EventMessageID:         "",
			}
			if payload.RoundID != sharedtypes.RoundID(uuid.Nil) {
				roundForError, getRoundErr := s.RoundDB.GetRound(ctx, payload.GuildID, payload.RoundID)
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
			// Return failure payload without error - this is expected business logic failure
			return RoundOperationResult{Failure: failurePayload}, nil
		}

		// Determine if Join is Late
		s.logger.InfoContext(ctx, "Fetching round details to determine if join is late",
			attr.ExtractCorrelationID(ctx),
			attr.RoundID("round_id", payload.RoundID),
		)
		round, err := s.RoundDB.GetRound(ctx, payload.GuildID, payload.RoundID)
		if err != nil {
			s.logger.InfoContext(ctx, "Failed to fetch round during join validation - returning failure payload",
				attr.ExtractCorrelationID(ctx),
				attr.RoundID("round_id", payload.RoundID),
				attr.String("user_id", string(payload.UserID)),
				attr.Error(err),
			)
			failurePayload := &roundevents.RoundParticipantJoinErrorPayload{
				ParticipantJoinRequest: &payload,
				Error:                  fmt.Sprintf("failed to fetch round details: %v", err),
				EventMessageID:         "",
			}
			if round != nil {
				failurePayload.EventMessageID = round.EventMessageID
			}

			// Return failure payload without error - this is expected business logic failure (round not found)
			return RoundOperationResult{Failure: failurePayload}, nil
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
			tagLookupPayload := &roundevents.TagLookupRequestPayload{
				RoundID:          payload.RoundID,
				UserID:           payload.UserID,
				Response:         payload.Response,
				OriginalResponse: payload.Response,
				JoinedLate:       payload.JoinedLate,
			}
			s.logger.InfoContext(ctx, "Validation successful for Accept/Tentative - Returning TagLookupPayload (pointer)",
				attr.ExtractCorrelationID(ctx),
				attr.Any("returning_payload", tagLookupPayload),
			)
			return RoundOperationResult{Success: tagLookupPayload}, nil

		case roundtypes.ResponseDecline:
			s.logger.InfoContext(ctx, "Validation successful for Decline - Returning ParticipantJoinRequestPayload (pointer)",
				attr.ExtractCorrelationID(ctx),
				attr.Any("returning_payload", payload),
			)
			return RoundOperationResult{Success: &payload}, nil

		default:
			// Handle unexpected response types
			s.logger.InfoContext(ctx, "Unexpected response type in validated payload - returning failure payload",
				attr.ExtractCorrelationID(ctx),
				attr.RoundID("round_id", payload.RoundID),
				attr.String("user_id", string(payload.UserID)),
				attr.String("response", string(payload.Response)),
			)
			failurePayload := &roundevents.RoundParticipantJoinErrorPayload{
				ParticipantJoinRequest: &payload,
				Error:                  fmt.Sprintf("unexpected response type: %s", payload.Response),
				EventMessageID:         "",
			}
			if round != nil {
				failurePayload.EventMessageID = round.EventMessageID
			}

			// Return failure payload without error - this is expected business logic failure
			return RoundOperationResult{Failure: failurePayload}, nil
		}
	})

	return result, err
}

// ParticipantRemoval handles removing a participant from a round if they select the same RSVP response
func (s *RoundService) ParticipantRemoval(ctx context.Context, payload roundevents.ParticipantRemovalRequestPayload) (RoundOperationResult, error) {
	result, err := s.serviceWrapper(ctx, "ParticipantRemoval", payload.RoundID, func(ctx context.Context) (RoundOperationResult, error) {
		s.logger.InfoContext(ctx, "Processing participant removal",
			attr.RoundID("round_id", payload.RoundID),
			attr.String("user_id", string(payload.UserID)),
		)

		// Get round details for EventMessageID before removal
		round, err := s.RoundDB.GetRound(ctx, payload.GuildID, payload.RoundID)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to fetch round during participant removal",
				attr.RoundID("round_id", payload.RoundID),
				attr.String("user_id", string(payload.UserID)),
				attr.Error(err),
			)
			failurePayload := &roundevents.ParticipantRemovalErrorPayload{
				RoundID: payload.RoundID,
				UserID:  payload.UserID,
				Error:   fmt.Sprintf("failed to fetch round details: %v", err),
			}
			return RoundOperationResult{Failure: failurePayload}, nil
		}

		// Remove participant and get updated participants list
		updatedParticipants, err := s.RoundDB.RemoveParticipant(ctx, payload.GuildID, payload.RoundID, payload.UserID)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to remove participant from DB",
				attr.RoundID("round_id", payload.RoundID),
				attr.String("user_id", string(payload.UserID)),
				attr.Error(err),
			)
			failurePayload := &roundevents.ParticipantRemovalErrorPayload{
				RoundID: payload.RoundID,
				UserID:  payload.UserID,
				Error:   fmt.Sprintf("failed to remove participant: %v", err),
			}
			return RoundOperationResult{Failure: failurePayload}, nil
		}

		// Categorize participants after removal
		accepted, declined, tentative := categorizeParticipants(updatedParticipants)

		// Prepare success payload
		removedPayload := &roundevents.ParticipantRemovedPayload{
			RoundID:               payload.RoundID,
			UserID:                payload.UserID,
			AcceptedParticipants:  accepted,
			DeclinedParticipants:  declined,
			TentativeParticipants: tentative,
			EventMessageID:        round.EventMessageID,
		}

		s.logger.InfoContext(ctx, "Participant removal successful",
			attr.RoundID("round_id", payload.RoundID),
			attr.String("user_id", string(payload.UserID)),
			attr.Int("accepted_count", len(accepted)),
			attr.Int("declined_count", len(declined)),
			attr.Int("tentative_count", len(tentative)),
		)

		return RoundOperationResult{Success: removedPayload}, nil
	})

	return result, err
}

// UpdateParticipantStatus is the main entry point - delegates to specific handlers
func (s *RoundService) UpdateParticipantStatus(ctx context.Context, payload roundevents.ParticipantJoinRequestPayload) (RoundOperationResult, error) {
	result, err := s.serviceWrapper(ctx, "UpdateParticipantStatus", payload.RoundID, func(ctx context.Context) (RoundOperationResult, error) {
		s.logger.InfoContext(ctx, "Processing participant status update",
			attr.RoundID("round_id", payload.RoundID),
			attr.String("user_id", string(payload.UserID)),
			attr.String("response", string(payload.Response)),
			attr.Any("tag_number", payload.TagNumber),
			attr.Any("joined_late", payload.JoinedLate),
		)

		// Route to appropriate handler based on payload characteristics
		switch {
		case payload.TagNumber != nil && (payload.Response == roundtypes.ResponseAccept || payload.Response == roundtypes.ResponseTentative):
			return s.updateParticipantWithTag(ctx, payload)
		case payload.TagNumber == nil && (payload.Response == roundtypes.ResponseAccept || payload.Response == roundtypes.ResponseTentative):
			return s.updateParticipantWithoutTag(ctx, payload)
		case payload.Response == roundtypes.ResponseDecline:
			return s.updateParticipantDecline(ctx, payload)
		default:
			return s.handleUnknownResponse(ctx, payload)
		}
	})

	return result, err
}

// updateParticipantWithTag handles Accept/Tentative with tag (from tag lookup "found")
func (s *RoundService) updateParticipantWithTag(ctx context.Context, payload roundevents.ParticipantJoinRequestPayload) (RoundOperationResult, error) {
	s.logger.InfoContext(ctx, "Updating participant with tag",
		attr.RoundID("round_id", payload.RoundID),
		attr.String("user_id", string(payload.UserID)),
		attr.Int("tag_number", int(*payload.TagNumber)),
	)

	participant := roundtypes.Participant{
		UserID:    payload.UserID,
		Response:  payload.Response,
		TagNumber: payload.TagNumber,
		Score:     nil,
	}

	return s.updateParticipantInDB(ctx, payload, participant)
}

// updateParticipantWithoutTag handles Accept/Tentative without tag (from tag lookup "not found")
func (s *RoundService) updateParticipantWithoutTag(ctx context.Context, payload roundevents.ParticipantJoinRequestPayload) (RoundOperationResult, error) {
	s.logger.InfoContext(ctx, "Updating participant without tag (tag not found)",
		attr.RoundID("round_id", payload.RoundID),
		attr.String("user_id", string(payload.UserID)),
	)

	participant := roundtypes.Participant{
		UserID:    payload.UserID,
		Response:  payload.Response,
		TagNumber: nil, // Tag lookup returned not found
		Score:     nil,
	}

	return s.updateParticipantInDB(ctx, payload, participant)
}

// updateParticipantDecline handles decline responses
func (s *RoundService) updateParticipantDecline(ctx context.Context, payload roundevents.ParticipantJoinRequestPayload) (RoundOperationResult, error) {
	s.logger.InfoContext(ctx, "Updating participant with decline",
		attr.RoundID("round_id", payload.RoundID),
		attr.String("user_id", string(payload.UserID)),
	)

	participant := roundtypes.Participant{
		UserID:    payload.UserID,
		Response:  payload.Response,
		TagNumber: nil, // No tag needed for decline
		Score:     nil,
	}

	return s.updateParticipantInDB(ctx, payload, participant)
}

// handleUnknownResponse handles unexpected response types
func (s *RoundService) handleUnknownResponse(ctx context.Context, payload roundevents.ParticipantJoinRequestPayload) (RoundOperationResult, error) {
	s.logger.ErrorContext(ctx, "Unknown response type",
		attr.RoundID("round_id", payload.RoundID),
		attr.String("user_id", string(payload.UserID)),
		attr.String("response", string(payload.Response)),
	)

	failurePayload := &roundevents.RoundParticipantJoinErrorPayload{
		ParticipantJoinRequest: &payload,
		Error:                  fmt.Sprintf("unknown response type: %s", payload.Response),
		EventMessageID:         "",
	}
	return RoundOperationResult{Failure: failurePayload}, nil
}

// updateParticipantInDB handles the common DB operations and response construction
func (s *RoundService) updateParticipantInDB(ctx context.Context, payload roundevents.ParticipantJoinRequestPayload, participant roundtypes.Participant) (RoundOperationResult, error) {
	// Get round details for EventMessageID
	round, err := s.RoundDB.GetRound(ctx, payload.GuildID, payload.RoundID)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to fetch round details",
			attr.RoundID("round_id", payload.RoundID),
			attr.String("user_id", string(payload.UserID)),
			attr.Error(err),
		)
		failurePayload := &roundevents.RoundParticipantJoinErrorPayload{
			ParticipantJoinRequest: &payload,
			Error:                  fmt.Sprintf("failed to fetch round details: %v", err),
			EventMessageID:         "",
		}
		return RoundOperationResult{Failure: failurePayload}, nil
	}

	// Update participant in database
	updatedParticipants, err := s.RoundDB.UpdateParticipant(ctx, payload.GuildID, payload.RoundID, participant)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to update participant in DB",
			attr.RoundID("round_id", payload.RoundID),
			attr.String("user_id", string(payload.UserID)),
			attr.Error(err),
		)
		failurePayload := &roundevents.RoundParticipantJoinErrorPayload{
			ParticipantJoinRequest: &payload,
			Error:                  fmt.Sprintf("failed to update participant in DB: %v", err),
			EventMessageID:         round.EventMessageID,
		}
		return RoundOperationResult{Failure: failurePayload}, nil
	}

	// Categorize participants and build success response
	accepted, declined, tentative := categorizeParticipants(updatedParticipants)

	isLateJoin := false
	if payload.JoinedLate != nil {
		isLateJoin = *payload.JoinedLate
	}

	joinedPayload := &roundevents.ParticipantJoinedPayload{
		RoundID:               payload.RoundID,
		AcceptedParticipants:  accepted,
		DeclinedParticipants:  declined,
		TentativeParticipants: tentative,
		EventMessageID:        round.EventMessageID,
		JoinedLate:            &isLateJoin,
	}

	return RoundOperationResult{Success: joinedPayload}, nil
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
