package roundservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// CheckParticipantStatus checks if this is a toggle action or a new join
func (s *RoundService) CheckParticipantStatus(ctx context.Context, msg *message.Message) error {
	correlationID, payload, err := eventutil.UnmarshalPayload[roundevents.ParticipantJoinRequestPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal ParticipantJoinRequestPayload: %w", err)
	}

	slog.Debug("Unmarshalled payload in CheckParticipantStatus",
		slog.Any("correlation_id", correlationID),
		slog.Any("payload", payload))

	// Check if the user is already a participant
	participant, err := s.RoundDB.GetParticipant(ctx, payload.RoundID, string(payload.UserID))
	if err != nil {
		slog.Error("Failed to get participant's current status",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err))
		return s.publishParticipantJoinError(msg, payload, err)
	}

	currentStatus := ""
	if participant != nil {
		currentStatus = string(participant.Response)
	}

	// Handle toggle removal (if the same status is clicked again)
	if currentStatus == string(payload.Response) {
		slog.Info("Toggle action detected - publishing removal request",
			slog.String("correlation_id", correlationID),
			slog.String("status", currentStatus))

		// Publish a removal request event
		if err := s.publishEvent(msg, roundevents.RoundParticipantRemovalRequest, roundevents.ParticipantRemovalRequestPayload{
			RoundID: payload.RoundID,
			UserID:  payload.UserID,
		}); err != nil {
			slog.Error("Failed to publish removal request",
				slog.String("correlation_id", correlationID),
				slog.Any("error", err))
			return err
		}
	} else {
		// Late Join Detection
		isLateJoin := payload.JoinedLate != nil && *payload.JoinedLate

		slog.Debug("Publishing validation request",
			slog.String("correlation_id", correlationID),
			slog.Bool("joined_late", isLateJoin),
			slog.Any("payload", payload))

		// Publish validation request, ensuring `JoinedLate` is passed correctly
		if err := s.publishEvent(msg, roundevents.RoundParticipantJoinValidationRequest, roundevents.ParticipantJoinRequestPayload{
			RoundID:    payload.RoundID,
			UserID:     payload.UserID,
			Response:   payload.Response,
			JoinedLate: payload.JoinedLate,
		}); err != nil {
			slog.Error("Failed to publish validation request",
				slog.String("correlation_id", correlationID),
				slog.Any("error", err))
			return err
		}
	}

	return nil
}

// ValidateParticipantJoinRequest validates the participant join request and publishes the next step.
func (s *RoundService) ValidateParticipantJoinRequest(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.ParticipantJoinRequestPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal ParticipantJoinRequestPayload: %w", err)
	}

	isLateJoin := eventPayload.JoinedLate != nil && *eventPayload.JoinedLate
	slog.Debug("Validating participant join request",
		slog.Bool("joined_late", isLateJoin),
		slog.Any("payload", eventPayload))

	// Validation checks
	if eventPayload.RoundID == 0 {
		return s.publishParticipantJoinError(msg, eventPayload, fmt.Errorf("round ID cannot be zero"))
	}

	if eventPayload.UserID == "" {
		return s.publishParticipantJoinError(msg, eventPayload, fmt.Errorf("participant Discord ID cannot be empty"))
	}

	payloadBytes, err := json.Marshal(eventPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create a Watermill message
	joinValidatedMsg := message.NewMessage(watermill.NewUUID(), payloadBytes)

	// Publish using EventBus
	if err := s.EventBus.Publish(roundevents.RoundParticipantJoinValidated, joinValidatedMsg); err != nil {
		slog.Error("🔥 Failed to publish round.participant.join.validated event", slog.Any("error", err))
		return fmt.Errorf("failed to publish round.participant.join.validated event: %w", err)
	}

	// Log if it's a late join
	if isLateJoin {
		slog.Info("Late join request validated",
			slog.Int64("round_id", int64(eventPayload.RoundID)),
			slog.String("user_id", string(eventPayload.UserID)))
	}

	return nil
}

// HandleParticipantRemoval handles removing a participant from a round.
func (s *RoundService) ParticipantRemoval(ctx context.Context, msg *message.Message) error {
	correlationID, eventPayload, err := eventutil.UnmarshalPayload[roundevents.ParticipantRemovalRequestPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal ParticipantRemovalRequestPayload: %w", err)
	}

	slog.Info("Removing participant from round",
		slog.String("correlation_id", correlationID),
		slog.Int64("round_id", int64(eventPayload.RoundID)),
		slog.String("user_id", string(eventPayload.UserID)))

	// Retrieve EventMessageID
	eventMessageID, err := s.getEventMessageID(ctx, eventPayload.RoundID)
	if err != nil {
		return err
	}

	// Get current response before removing (for event publishing)
	participant, err := s.RoundDB.GetParticipant(ctx, eventPayload.RoundID, string(eventPayload.UserID))
	if err != nil {
		slog.Error("Failed to get participant response before removal",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err))
		return fmt.Errorf("failed to get participant response before removal: %w", err)
	}

	// Determine the response to include in the event
	var response roundtypes.Response
	if participant == nil {
		response = roundtypes.Response("") // User was not in the round
	} else {
		response = roundtypes.Response(participant.Response)
	}

	// Remove participant from database
	if err := s.RoundDB.RemoveParticipant(ctx, eventPayload.RoundID, string(eventPayload.UserID)); err != nil {
		slog.Error("Failed to remove participant from database",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err))
		return fmt.Errorf("failed to remove participant from database: %w", err)
	}

	// Publish participant removed event
	if err := s.publishEvent(msg, roundevents.RoundParticipantRemoved, roundevents.ParticipantRemovedPayload{
		RoundID:        eventPayload.RoundID,
		UserID:         eventPayload.UserID,
		Response:       response,
		EventMessageID: eventMessageID, // ✅ Now included!
	}); err != nil {
		slog.Error(
			"Failed to publish participant.removed event",
			map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("failed to publish participant.removed event: %w", err)
	}

	slog.Info("Participant removed successfully",
		slog.String("correlation_id", correlationID))

	return nil
}

// ParticipantTagFound handles the round.tag.number.found event.
// ParticipantTagFound handles the round.tag.number.found event.
func (s *RoundService) ParticipantTagFound(ctx context.Context, msg *message.Message) error {
	correlationID, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundTagNumberFoundPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal RoundTagNumberFoundPayload: %w", err)
	}

	slog.Info("Received ParticipantTagFound event",
		slog.String("correlation_id", correlationID),
		slog.String("user_id", string(eventPayload.UserID)),
		slog.Any("tag_number", eventPayload.TagNumber))

	roundID := eventPayload.RoundID

	// Retrieve EventMessageID
	eventMessageID, err := s.getEventMessageID(ctx, roundID)
	if err != nil {
		return err
	}

	// Ensure tag number is properly assigned
	var tagNumber *int
	if eventPayload.TagNumber != nil {
		tagNumber = eventPayload.TagNumber
	} else {
		slog.Warn("Tag number is missing in event payload", slog.String("user_id", string(eventPayload.UserID)))
	}

	// Create a new participant with the found tag number
	participant := roundtypes.Participant{
		UserID:    string(eventPayload.UserID),
		Response:  string(roundtypes.ResponseAccept),
		TagNumber: tagNumber, // Ensure this is not nil if it should have a value
		Score:     nil,
	}

	// Update participant in the database
	updatedParticipants, err := s.RoundDB.UpdateParticipant(ctx, roundID, participant)
	if err != nil {
		slog.Error("Failed to update participant in database",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return err
	}

	slog.Info("Updated participants list from database",
		slog.Any("updated_participants", updatedParticipants),
	)

	// Categorize participants
	var accepted, declined, tentative []roundtypes.Participant
	for _, p := range updatedParticipants {
		switch roundtypes.Response(p.Response) {
		case roundtypes.ResponseAccept:
			accepted = append(accepted, p)
		case roundtypes.ResponseDecline:
			declined = append(declined, p)
		case roundtypes.ResponseTentative:
			tentative = append(tentative, p)
		}
	}

	slog.Info("Categorized participants",
		slog.Any("accepted", accepted),
		slog.Any("declined", declined),
		slog.Any("tentative", tentative),
	)

	// Publish updated participant list
	joinedPayload := roundevents.ParticipantJoinedPayload{
		RoundID:               roundID,
		AcceptedParticipants:  accepted,
		DeclinedParticipants:  declined,
		TentativeParticipants: tentative,
		EventMessageID:        eventMessageID,
	}

	payloadBytes, err := json.Marshal(joinedPayload)
	if err != nil {
		slog.Error("Failed to marshal participant joined payload",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to marshal participant joined payload: %w", err)
	}

	newMessage := message.NewMessage(watermill.NewUUID(), payloadBytes)
	if err := s.EventBus.Publish(roundevents.RoundParticipantJoined, newMessage); err != nil {
		slog.Error("Failed to publish round.participant.joined event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to publish round.participant.joined event: %w", err)
	}

	slog.Info("Successfully published discord.round.participant.joined event",
		slog.Any("published_payload", joinedPayload),
	)

	return nil
}

// ParticipantTagNotFound handles the round.tag.number.notfound event.
func (s *RoundService) ParticipantTagNotFound(ctx context.Context, msg *message.Message) error {
	correlationID, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundTagNumberNotFoundPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal RoundTagNumberNotFoundPayload: %w", err)
	}

	slog.Info("Received ParticipantTagNotFound event",
		slog.String("correlation_id", correlationID))

	roundID := eventPayload.RoundID

	// Retrieve EventMessageID
	eventMessageID, err := s.getEventMessageID(ctx, roundID)
	if err != nil {
		return err
	}

	slog.Info("About to categorize participants 🚀") // Add this line

	// Create a new participant with no tag but with ACCEPT status
	participant := roundtypes.Participant{
		UserID:    string(eventPayload.UserID),
		Response:  string(roundtypes.ResponseAccept),
		TagNumber: nil, // No tag number
		Score:     nil,
	}

	// Get the updated participants list from the database
	updatedParticipants, err := s.RoundDB.UpdateParticipant(ctx, roundID, participant)
	if err != nil {
		slog.Error("Failed to update participant in database",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)

		joinRequestPayload := roundevents.ParticipantJoinRequestPayload{
			RoundID:    roundID,
			UserID:     eventPayload.UserID,
			Response:   roundtypes.ResponseAccept,
			JoinedLate: nil,
		}
		return s.publishParticipantJoinError(msg, joinRequestPayload, err)
	}

	// Log what's in the updated participants list
	slog.Info("Updated participants list from database",
		slog.Any("updated_participants", updatedParticipants),
	)

	// Assign participants to respective lists
	var accepted, declined, tentative []roundtypes.Participant
	for _, p := range updatedParticipants {
		switch roundtypes.Response(p.Response) {
		case roundtypes.ResponseAccept:
			accepted = append(accepted, p)
		case roundtypes.ResponseDecline:
			declined = append(declined, p)
		case roundtypes.ResponseTentative:
			tentative = append(tentative, p)
		}
	}

	slog.Info("Categorized participants",
		slog.Any("accepted", accepted),
		slog.Any("declined", declined),
		slog.Any("tentative", tentative),
	)

	// Construct the payload for the participant joined event
	joinedPayload := roundevents.ParticipantJoinedPayload{
		RoundID:               roundID,
		AcceptedParticipants:  accepted,
		DeclinedParticipants:  declined,
		TentativeParticipants: tentative,
		EventMessageID:        eventMessageID,
	}

	// *** ADDED LOGGING HERE ***
	slog.Info("Before marshalling joinedPayload",
		slog.Any("accepted", accepted),
		slog.Any("declined", declined),
		slog.Any("tentative", tentative),
	)

	// Marshal the payload
	payloadBytes, err := json.Marshal(joinedPayload)
	if err != nil {
		slog.Error("Failed to marshal participant joined payload",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to marshal participant joined payload: %w", err)
	}
	slog.Info("ParticipantJoinedPayload being sent",
		slog.Any("payload", joinedPayload),
	)

	// Create a new Watermill message
	newMessage := message.NewMessage(watermill.NewUUID(), payloadBytes)

	// Publish the event
	if err := s.EventBus.Publish(roundevents.RoundParticipantJoined, newMessage); err != nil {
		slog.Error("Failed to publish discord.round.participant.joined event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to publish round.participant.joined event: %w", err)
	}

	slog.Info("Successfully published discord.round.participant.joined event",
		slog.Any("published_payload", joinedPayload),
	)

	return nil
}

// HandleDecline logs the participant's decline response and updates the database.
func (s *RoundService) HandleDecline(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.ParticipantDeclinedPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal ParticipantDeclinedPayload: %w", err)
	}

	eventMessageID, err := s.getEventMessageID(ctx, eventPayload.RoundID)
	if err != nil {
		return err
	}

	if err := s.publishEvent(msg, roundevents.RoundParticipantDeclinedResponse, roundevents.ParticipantDeclinedPayload{
		RoundID:        eventPayload.RoundID,
		UserID:         eventPayload.UserID,
		EventMessageID: eventMessageID,
	}); err != nil {
		return fmt.Errorf("failed to publish participant declined notification for round %d: %w", eventPayload.RoundID, err)
	}

	return nil
}

// publishParticipantJoinError publishes a round.participant.join.error event.
func (s *RoundService) publishParticipantJoinError(msg *message.Message, input roundevents.ParticipantJoinRequestPayload, err error) error {
	eventMessageID, getErr := s.getEventMessageID(context.Background(), input.RoundID)
	if getErr != nil {
		slog.Error("Failed to retrieve EventMessageID for error event", slog.Any("error", getErr))
		eventMessageID = "" // Fallback to empty
	}

	payload := roundevents.RoundParticipantJoinErrorPayload{
		ParticipantJoinRequest: &input,
		Error:                  err.Error(),
		EventMessageID:         eventMessageID,
	}

	if pubErr := s.publishEvent(msg, roundevents.RoundParticipantJoinError, payload); pubErr != nil {
		return fmt.Errorf("failed to publish round.participant.join.error event: %w, original error: %w", pubErr, err)
	}

	return err
}
