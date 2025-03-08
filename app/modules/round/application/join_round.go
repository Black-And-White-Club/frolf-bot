package roundservice

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/app/shared/logging"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
)

// -- Service Functions for JoinRound Flow --

// ValidateParticipantJoinRequest validates the participant join request.
func (s *RoundService) ValidateParticipantJoinRequest(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.ParticipantJoinRequestPayload](msg, s.logger)
	if err != nil {
		return s.publishParticipantJoinError(msg, eventPayload, fmt.Errorf("invalid payload: %w", err))
	}

	if eventPayload.RoundID == 0 { // Check if RoundID is zero
		err := fmt.Errorf("round ID cannot be zero")
		return s.publishParticipantJoinError(msg, eventPayload, err)
	}
	if eventPayload.DiscordID == "" { // Changed to DiscordID
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

	// Convert int64 RoundID to string
	roundIDStr := strconv.FormatInt(eventPayload.ParticipantJoinRequestPayload.RoundID, 10)

	// Set the RoundID in the metadata for later use.
	msg.Metadata.Set("RoundID", roundIDStr) // Use the converted string

	// Publish a "round.tag.number.request" event to get tag number in tag_retrieval.go
	// Correctly construct the payload for the event
	if err := s.publishEvent(msg, roundevents.RoundTagNumberRequest, roundevents.TagNumberRequestPayload{
		DiscordID: eventPayload.ParticipantJoinRequestPayload.DiscordID,
	}); err != nil {
		logging.LogErrorWithMetadata(ctx, s.logger, msg, "Failed to publish round.tag.number.request event", map[string]interface{}{"error": err.Error()})
		return fmt.Errorf("failed to publish round.tag.number.request event: %w", err)
	}

	logging.LogInfoWithMetadata(ctx, s.logger, msg, "Published round.tag.number.request event", map[string]interface{}{"user_id": eventPayload.ParticipantJoinRequestPayload.DiscordID})

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

	// Convert roundtypes.Response to rounddb.Response
	dbResponse := rounddb.Response(eventPayload.Response)

	// Convert roundtypes.RoundParticipant.TagNumber to *int
	dbTagNumber := &eventPayload.TagNumber

	// Convert roundtypes.RoundParticipant to rounddb.Participant
	dbParticipant := rounddb.Participant{
		DiscordID: eventPayload.DiscordID,
		Response:  dbResponse,
		TagNumber: dbTagNumber,
		Score:     nil, // Score is initialized to nil
	}

	if err = s.RoundDB.UpdateParticipant(ctx, roundID, dbParticipant); err != nil {
		s.logger.Error("Failed to update participant in database",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return s.publishParticipantJoinError(msg, roundevents.ParticipantJoinRequestPayload{}, err)
	}

	s.logger.Info("ParticipantJoinRequest event processed", slog.String("correlation_id", correlationID))
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

	// Convert roundtypes.RoundParticipant to rounddb.Participant
	dbParticipant := rounddb.Participant{
		DiscordID: eventPayload.DiscordID,
		Response:  rounddb.Response(roundtypes.ResponseAccept), // Assuming they are joining the round
		TagNumber: &[]int{0}[0],                                // Set special value to indicate no tag
		Score:     nil,
	}

	if err = s.RoundDB.UpdateParticipant(ctx, roundID, dbParticipant); err != nil {
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
		TagNumber:   0,
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
