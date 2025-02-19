package roundservice

import (
	"context"
	"fmt"
	"log/slog"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/Black-And-White-Club/frolf-bot/app/shared/logging"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// -- Service Functions for JoinRound Flow --

// ValidateParticipantJoinRequest validates the participant join request.
func (s *RoundService) ValidateParticipantJoinRequest(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.ParticipantJoinRequestPayload](msg, s.logger)
	if err != nil {
		return s.publishParticipantJoinError(msg, eventPayload, fmt.Errorf("invalid payload: %w", err))
	}

	if eventPayload.RoundID == "" {
		err := fmt.Errorf("round ID cannot be empty")
		return s.publishParticipantJoinError(msg, eventPayload, err)
	}
	if eventPayload.Participant == "" {
		err := fmt.Errorf("participant Discord ID cannot be empty")
		return s.publishParticipantJoinError(msg, eventPayload, err)
	}

	// If validation passes, publish a "round.participant.join.validated" event
	if err := s.publishEvent(msg, roundevents.RoundParticipantJoinValidated, roundevents.ParticipantJoinValidatedPayload{
		ParticipantJoinRequestPayload: eventPayload,
	}); err != nil {
		logging.LogErrorWithMetadata(ctx, s.logger, msg, "Failed to publish round.participant.join.validated event", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("failed to publish round.participant.join.validated event: %w", err)
	}

	logging.LogInfoWithMetadata(ctx, s.logger, msg, "Participant join request validated", map[string]interface{}{"round_id": eventPayload.RoundID})
	return nil
}

// CheckParticipantTag initiates the tag number retrieval process for the participant.
func (s *RoundService) CheckParticipantTag(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.ParticipantJoinValidatedPayload](msg, s.logger)
	if err != nil {
		return s.publishParticipantJoinError(msg, eventPayload.ParticipantJoinRequestPayload, fmt.Errorf("invalid payload: %w", err))
	}

	// Set the RoundID in the metadata for later use.
	msg.Metadata.Set("RoundID", eventPayload.ParticipantJoinRequestPayload.RoundID)

	// Publish a "round.tag.number.request" event to get tag number in tag_retrieval.go
	// Correctly construct the payload for the event
	if err := s.publishEvent(msg, roundevents.RoundTagNumberRequest, roundevents.TagNumberRequestPayload{
		DiscordID: eventPayload.ParticipantJoinRequestPayload.Participant,
	}); err != nil {
		logging.LogErrorWithMetadata(ctx, s.logger, msg, "Failed to publish round.tag.number.request event", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("failed to publish round.tag.number.request event: %w", err)
	}

	logging.LogInfoWithMetadata(ctx, s.logger, msg, "Published round.tag.number.request event", map[string]interface{}{"user_id": eventPayload.ParticipantJoinRequestPayload.Participant})

	return nil
}

// ParticipantTagFound handles the round.tag.number.found event.
func (s *RoundService) ParticipantTagFound(ctx context.Context, msg *message.Message) error {
	correlationID, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundTagNumberFoundPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal RoundTagNumberFoundPayload: %w", err)
	}

	roundID := msg.Metadata.Get("RoundID")
	if roundID == "" {
		return fmt.Errorf("round ID not found in metadata")
	}

	// Update the participant's response and tag number in the database
	participant := roundtypes.RoundParticipant{
		DiscordID: eventPayload.DiscordID,
		Response:  roundtypes.ResponseAccept, // Assuming they are joining the round
		TagNumber: eventPayload.TagNumber,    // Now we have the tag number
	}

	if err = s.RoundDB.UpdateParticipant(ctx, roundID, participant); err != nil {
		s.logger.Error("Failed to update participant in database",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return s.publishParticipantJoinError(msg, roundevents.ParticipantJoinRequestPayload{}, err)
	}

	// Publish a "round.participant.joined" event
	if err := s.publishEvent(msg, roundevents.ParticipantJoined, roundevents.ParticipantJoinedPayload{
		RoundID:     roundID,
		Participant: eventPayload.DiscordID,
		Response:    "accept",
		TagNumber:   eventPayload.TagNumber,
	}); err != nil {
		logging.LogErrorWithMetadata(ctx, s.logger, msg, "Failed to publish round.participant.joined event", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("failed to publish round.participant.joined event: %w", err)
	}

	logging.LogInfoWithMetadata(ctx, s.logger, msg, "Participant joined round and updated in database", map[string]interface{}{
		"correlation_id": correlationID,
		"round_id":       roundID,
		"participant_id": eventPayload.DiscordID,
		"tag_number":     eventPayload.TagNumber,
	})

	return nil
}

// ParticipantTagNotFound handles the round.tag.number.notfound event.
func (s *RoundService) ParticipantTagNotFound(ctx context.Context, msg *message.Message) error {
	correlationID, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundTagNumberNotFoundPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal RoundTagNumberNotFoundPayload: %w", err)
	}

	// Get RoundID from message metadata
	roundID := msg.Metadata.Get("RoundID")
	if roundID == "" {
		return fmt.Errorf("RoundID not found in message metadata")
	}

	// Update the participant's response in the database (without a tag number)
	participant := roundtypes.RoundParticipant{
		DiscordID: eventPayload.DiscordID,
		Response:  roundtypes.ResponseAccept, // Assuming they are joining the round
		TagNumber: 0,                         // Set special value to indicate no tag
	}

	if err = s.RoundDB.UpdateParticipant(ctx, roundID, participant); err != nil {
		s.logger.Error("Failed to update participant in database",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return s.publishParticipantJoinError(msg, roundevents.ParticipantJoinRequestPayload{}, err) // Error event
	}

	// Publish a "round.participant.joined" event (without a tag number)
	if err := s.publishEvent(msg, roundevents.ParticipantJoined, roundevents.ParticipantJoinedPayload{
		RoundID:     roundID,
		Participant: eventPayload.DiscordID,
		Response:    "accept",
	}); err != nil {
		logging.LogErrorWithMetadata(ctx, s.logger, msg, "Failed to publish round.participant.joined event", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("failed to publish round.participant.joined event: %w", err)
	}

	logging.LogInfoWithMetadata(ctx, s.logger, msg, "Participant joined round (without tag) and updated in database", map[string]interface{}{
		"correlation_id": correlationID,
		"round_id":       roundID,
		"participant_id": eventPayload.DiscordID,
	})

	return nil
}

// publishParticipantJoinError publishes a round.participant.join.error event.
func (s *RoundService) publishParticipantJoinError(msg *message.Message, input roundevents.ParticipantJoinRequestPayload, err error) error {
	payload := roundevents.RoundParticipantJoinErrorPayload{
		CorrelationID:          middleware.MessageCorrelationID(msg),
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
