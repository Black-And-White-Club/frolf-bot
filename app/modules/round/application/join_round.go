package roundservice

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/Black-And-White-Club/frolf-bot/app/shared/logging"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
)

// -- Service Functions for JoinRound Flow --

// ValidateParticipantJoinRequest validates the participant join request and publishes the next step.
func (s *RoundService) ValidateParticipantJoinRequest(ctx context.Context, msg *message.Message) error {
	// Unmarshal the payload from the message
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.ParticipantJoinRequestPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal ParticipantJoinRequestPayload: %w", err)
	}

	// Log the entire unmarshalled payload for debugging
	s.logger.Debug("Unmarshalled payload",
		slog.Any("payload", eventPayload))

	// Validate RoundID
	if eventPayload.RoundID == 0 {
		return s.publishParticipantJoinError(msg, eventPayload, fmt.Errorf("round ID cannot be zero"))
	}

	// Validate UserID
	if eventPayload.UserID == "" {
		return s.publishParticipantJoinError(msg, eventPayload, fmt.Errorf("participant Discord ID cannot be empty"))
	}

	// Log the validated RoundID and UserID
	s.logger.Debug("Validated payload",
		slog.Int64("round_id", int64(eventPayload.RoundID)),
		slog.String("user_id", string(eventPayload.UserID)))

	// Determine the next step based on the RSVP response
	switch eventPayload.Response {
	case roundtypes.ResponseDecline:
		// Publish a decline event
		if err := s.publishEvent(msg, roundevents.RoundParticipantDeclined, roundevents.ParticipantDeclinedPayload{
			RoundID: eventPayload.RoundID,
			UserID:  eventPayload.UserID,
		}); err != nil {
			return fmt.Errorf("failed to publish round.participant.declined event: %w", err)
		}
		logging.LogInfoWithMetadata(ctx, s.logger, msg, "Participant declined", map[string]interface{}{
			"round_id": eventPayload.RoundID,
			"user_id":  eventPayload.UserID,
		})

	case roundtypes.ResponseAccept, roundtypes.ResponseTentative:
		// Publish a validated event for Accept or Tentative
		if err := s.publishEvent(msg, roundevents.RoundParticipantJoinValidated, eventPayload); err != nil {
			return fmt.Errorf("failed to publish round.participant.join.validated event: %w", err)
		}
		logging.LogInfoWithMetadata(ctx, s.logger, msg, "Participant join request validated", map[string]interface{}{
			"round_id": eventPayload.RoundID,
		})

		// Publish a tag number request event
		if err := s.publishEvent(msg, roundevents.RoundTagNumberRequest, roundevents.TagNumberRequestPayload{
			UserID: eventPayload.UserID,
		}); err != nil {
			return fmt.Errorf("failed to publish round.tag.number.request event: %w", err)
		}
		logging.LogInfoWithMetadata(ctx, s.logger, msg, "Published round.tag.number.request event", map[string]interface{}{
			"user_id": eventPayload.UserID,
		})

	default:
		return fmt.Errorf("unknown response type: %s", eventPayload.Response)
	}

	return nil
}

// HandleParticipantRemoval handles removing a participant from a round.
func (s *RoundService) ParticipantRemoval(ctx context.Context, msg *message.Message) error {
	correlationID, eventPayload, err := eventutil.UnmarshalPayload[roundevents.ParticipantRemovalRequestPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal ParticipantRemovalRequestPayload: %w", err)
	}

	s.logger.Info("Removing participant from round",
		slog.String("correlation_id", correlationID),
		slog.Int64("round_id", int64(eventPayload.RoundID)),
		slog.String("user_id", string(eventPayload.UserID)))

	// Get current response before removing (for the event)
	participant, err := s.RoundDB.GetParticipant(ctx, eventPayload.RoundID, string(eventPayload.UserID))
	if err != nil {
		s.logger.Error("Failed to get participant response before removal",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err))
		return fmt.Errorf("failed to get participant response before removal: %w", err)
	}

	// Get response value for the event
	var response roundtypes.Response
	if participant == nil {
		// They weren't in the round to begin with
		response = roundtypes.Response("")
	} else {
		response = roundtypes.Response(participant.Response)
	}

	// Remove participant from database
	if err := s.RoundDB.RemoveParticipant(ctx, eventPayload.RoundID, string(eventPayload.UserID)); err != nil {
		s.logger.Error("Failed to remove participant from database",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err))
		return fmt.Errorf("failed to remove participant from database: %w", err)
	}

	// Publish participant removed event
	if err := s.publishEvent(msg, roundevents.RoundParticipantRemoved, roundevents.ParticipantRemovedPayload{
		RoundID:  eventPayload.RoundID,
		UserID:   eventPayload.UserID,
		Response: response, // Include the response they were removed from
	}); err != nil {
		logging.LogErrorWithMetadata(ctx, s.logger, msg,
			"Failed to publish participant.removed event",
			map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("failed to publish participant.removed event: %w", err)
	}

	s.logger.Info("Participant removed successfully",
		slog.String("correlation_id", correlationID))
	return nil
}

// ParticipantTagFound handles the round.tag.number.found event.
func (s *RoundService) ParticipantTagFound(ctx context.Context, msg *message.Message) error {
	correlationID, eventPayload, err := eventutil.UnmarshalPayload[roundevents.ParticipantJoinRequestPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal ParticipantJoinRequestPayload: %w", err)
	}

	s.logger.Info("Received ParticipantJoinRequest event", slog.String("correlation_id", correlationID))

	roundID := eventPayload.RoundID

	// Convert response and tag number
	participant := roundtypes.Participant{
		UserID:    eventPayload.UserID,
		Response:  eventPayload.Response,
		TagNumber: eventPayload.TagNumber,
		Score:     nil, // Score is initialized to nil
	}

	// Update the participant in the database
	if err = s.RoundDB.UpdateParticipant(ctx, roundID, participant); err != nil {
		s.logger.Error("Failed to update participant in database",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return s.publishParticipantJoinError(msg, eventPayload, err)
	}

	// Publish the joined event
	if err := s.publishEvent(msg, roundevents.RoundParticipantJoined, roundevents.ParticipantJoinedPayload{
		RoundID: roundID,
		Participant: roundtypes.Participant{
			UserID:    eventPayload.UserID,
			Response:  eventPayload.Response,
			TagNumber: eventPayload.TagNumber,
			Score:     nil,
		},
	}); err != nil {
		logging.LogErrorWithMetadata(ctx, s.logger, msg, "Failed to publish round.participant.joined event", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("failed to publish round.participant.joined event: %w", err)
	}

	return nil
}

// CheckParticipantStatus checks if this is a toggle action or a new join
func (s *RoundService) CheckParticipantStatus(ctx context.Context, msg *message.Message) error {
	correlationID, payload, err := eventutil.UnmarshalPayload[roundevents.ParticipantJoinRequestPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal ParticipantJoinRequestPayload: %w", err)
	}

	s.logger.Debug("Unmarshalled payload in CheckParticipantStatus",
		slog.String("correlation_id", correlationID),
		slog.Any("payload", payload))

	// Get current status from database
	participant, err := s.RoundDB.GetParticipant(ctx, payload.RoundID, string(payload.UserID))
	if err != nil {
		s.logger.Error("Failed to get participant's current status",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err))
		return s.publishParticipantJoinError(msg, payload, err)
	}

	currentStatus := ""
	if participant != nil {
		currentStatus = string(participant.Response)
	}

	// Check if this is a toggle action
	if currentStatus == string(payload.Response) {
		s.logger.Info("Toggle action detected - publishing removal request",
			slog.String("correlation_id", correlationID),
			slog.String("status", currentStatus))

		// Publish a removal request event
		if err := s.publishEvent(msg, roundevents.RoundParticipantRemovalRequest, roundevents.ParticipantRemovalRequestPayload{
			RoundID: payload.RoundID,
			UserID:  payload.UserID,
		}); err != nil {
			s.logger.Error("Failed to publish removal request",
				slog.String("correlation_id", correlationID),
				slog.Any("error", err))
			return err
		}
	} else {
		// Not a toggle action, proceed with validation
		s.logger.Debug("Publishing validation request",
			slog.String("correlation_id", correlationID),
			slog.Any("payload", roundevents.ParticipantJoinRequestPayload{
				RoundID:  payload.RoundID,
				UserID:   payload.UserID,
				Response: payload.Response,
			}))

		if err := s.publishEvent(msg, roundevents.RoundParticipantJoinValidationRequest, roundevents.ParticipantJoinRequestPayload{
			RoundID:  payload.RoundID,
			UserID:   payload.UserID,
			Response: payload.Response,
		}); err != nil {
			s.logger.Error("Failed to publish validation request",
				slog.String("correlation_id", correlationID),
				slog.Any("error", err))
			return err
		}
	}

	return nil
}

// ParticipantTagNotFound handles the round.tag.number.notfound event.
func (s *RoundService) ParticipantTagNotFound(ctx context.Context, msg *message.Message) error {
	correlationID, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundTagNumberNotFoundPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal RoundTagNumberNotFoundPayload: %w", err)
	}

	// Get RoundID from message metadata
	roundIDStr := msg.Metadata.Get("RoundID")
	if roundIDStr == "" {
		return fmt.Errorf("RoundID not found in message metadata")
	}

	// Convert roundIDStr to int64
	roundID, err := strconv.ParseInt(roundIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid RoundID in message metadata: %w", err)
	}

	// Create a new participant with no tag
	participant := roundtypes.Participant{
		UserID:    eventPayload.UserID,
		Response:  roundtypes.ResponseAccept,
		TagNumber: nil, // No tag number
		Score:     nil,
	}

	if err = s.RoundDB.UpdateParticipant(ctx, roundtypes.ID(roundID), participant); err != nil {
		s.logger.Error("Failed to update participant in database",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return s.publishParticipantJoinError(msg, roundevents.ParticipantJoinRequestPayload{
			RoundID: roundtypes.ID(roundID),
			UserID:  eventPayload.UserID,
		}, err)
	}

	// Publish joined event with nil tag number
	if err := s.publishEvent(msg, roundevents.RoundParticipantJoined, roundevents.ParticipantJoinedPayload{
		RoundID: roundtypes.ID(roundID),
		Participant: roundtypes.Participant{
			UserID:    eventPayload.UserID,
			TagNumber: nil,
			Response:  roundtypes.ResponseAccept,
			Score:     nil,
		},
	}); err != nil {
		logging.LogErrorWithMetadata(ctx, s.logger, msg, "Failed to publish round.participant.joined event", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("failed to publish round.participant.joined event: %w", err)
	}

	return nil
}

// HandleDecline logs the participant's decline response and updates the database.
func (s *RoundService) HandleDecline(ctx context.Context, msg *message.Message) error {
	// Unmarshal the payload to get the event data
	correlationID, eventPayload, err := eventutil.UnmarshalPayload[roundevents.ParticipantDeclinedPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal ParticipantDeclinedPayload: %w", err)
	}

	// Create a rounddb.Participant object for the decline
	participant := roundtypes.Participant{
		UserID:    eventPayload.UserID,
		Response:  roundtypes.ResponseAccept,
		TagNumber: nil, // No tag number
		Score:     nil,
	}

	if err = s.RoundDB.UpdateParticipant(ctx, roundtypes.ID(eventPayload.RoundID), participant); err != nil {
		s.logger.Error("Failed to update participant in database",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return s.publishParticipantJoinError(msg, roundevents.ParticipantJoinRequestPayload{
			RoundID: roundtypes.ID(eventPayload.RoundID),
			UserID:  eventPayload.UserID,
		}, err)
	}

	// Publish an event to notify Discord to update the embed
	if err := s.publishEvent(nil, roundevents.RoundParticipantDeclinedResponse, roundevents.ParticipantDeclinedPayload{
		RoundID: eventPayload.RoundID,
		UserID:  eventPayload.UserID,
	}); err != nil {
		return fmt.Errorf("failed to publish participant declined notification for round %d: %w", eventPayload.RoundID, err)
	}

	return nil
}

// publishParticipantJoinError publishes a round.participant.join.error event.
func (s *RoundService) publishParticipantJoinError(msg *message.Message, input roundevents.ParticipantJoinRequestPayload, err error) error {
	payload := roundevents.RoundParticipantJoinErrorPayload{
		ParticipantJoinRequest: &input,
		Error:                  err.Error(),
	}

	if pubErr := s.publishEvent(msg, roundevents.RoundParticipantJoinError, payload); pubErr != nil {
		logging.LogErrorWithMetadata(context.Background(), s.logger, msg, "Failed to publish round.participant.join.error event", map[string]interface{}{
			"original_error": err.Error(),
		})
		return fmt.Errorf("failed to publish round.participant.join.error event: %w, original error: %w", pubErr, err)
	}

	return err
}
