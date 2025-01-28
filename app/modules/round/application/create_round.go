package roundservice

import (
	"context"
	"fmt"
	"strings"

	roundevents "github.com/Black-And-White-Club/tcr-bot/app/modules/round/domain/events"
	roundtypes "github.com/Black-And-White-Club/tcr-bot/app/modules/round/domain/types"
	roundutil "github.com/Black-And-White-Club/tcr-bot/app/modules/round/utils"
	"github.com/Black-And-White-Club/tcr-bot/app/shared/logging"
	"github.com/Black-And-White-Club/tcr-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// -- Service Functions for CreateRound Flow --

func (s *RoundService) ValidateRoundRequest(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundCreateRequestPayload](msg, s.logger)
	if err != nil {
		return s.publishRoundError(msg, &eventPayload, []error{fmt.Errorf("invalid payload: %w", err)})
	}

	// Validate the input using the validator
	input := roundtypes.CreateRoundInput{
		Title:        eventPayload.Title,
		Location:     eventPayload.Location,
		EventType:    eventPayload.EventType,
		StartTime:    eventPayload.DateTime,
		Participants: eventPayload.Participants,
	}

	if errs := s.roundValidator.ValidateRoundInput(input); len(errs) > 0 {
		logging.LogErrorWithMetadata(ctx, s.logger, msg, "Round input validation failed", map[string]interface{}{
			"errors": errs,
		})
		return s.publishRoundError(msg, &eventPayload, errs)
	}

	// If validation passes, publish a "round.validated" event
	if err := s.publishEvent(msg, roundevents.RoundValidated, roundevents.RoundValidatedPayload{
		RoundCreateRequestPayload: eventPayload,
	}); err != nil {
		logging.LogErrorWithMetadata(ctx, s.logger, msg, "Failed to publish round.validated event", map[string]interface{}{
			"error": err,
		})
		return fmt.Errorf("failed to publish round.validated event: %w", err)
	}

	logging.LogInfoWithMetadata(ctx, s.logger, msg, "Round input validated", map[string]interface{}{"title": eventPayload.Title})
	return nil
}

// ParseDateTime parses the date and time strings from the request.
func (s *RoundService) ParseDateTime(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundValidatedPayload](msg, s.logger)
	if err != nil {
		return s.publishRoundError(msg, &eventPayload.RoundCreateRequestPayload, []error{fmt.Errorf("invalid payload: %w", err)})
	}

	startTime, err := roundutil.NewDateTimeParser().ParseDateTime(eventPayload.RoundCreateRequestPayload.DateTime.Date + " " + eventPayload.RoundCreateRequestPayload.DateTime.Time)
	if err != nil {
		return s.publishRoundError(msg, &eventPayload.RoundCreateRequestPayload, []error{err})
	}

	// If parsing is successful, publish a "round.datetime.parsed" event
	if err := s.publishEvent(msg, roundevents.RoundDateTimeParsed, roundevents.RoundDateTimeParsedPayload{
		RoundCreateRequestPayload: eventPayload.RoundCreateRequestPayload,
		StartTime:                 startTime,
	}); err != nil {
		logging.LogErrorWithMetadata(ctx, s.logger, msg, "Failed to publish round.datetime.parsed event", map[string]interface{}{
			"error": err,
		})
		return fmt.Errorf("failed to publish round.datetime.parsed event: %w", err)
	}

	logging.LogInfoWithMetadata(ctx, s.logger, msg, "Date/time parsed successfully", map[string]interface{}{
		"start_time": startTime,
	})
	return nil
}

// CreateRoundEntity creates a Round entity from the request and parsed start time.
func (s *RoundService) CreateRoundEntity(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundDateTimeParsedPayload](msg, s.logger)
	if err != nil {
		return s.publishRoundError(msg, &eventPayload.RoundCreateRequestPayload, []error{fmt.Errorf("invalid payload: %w", err)})
	}

	round := createRoundEntity(eventPayload)

	// Publish "round.entity.created" event
	if err := s.publishEvent(msg, roundevents.RoundEntityCreated, roundevents.RoundEntityCreatedPayload{
		Round: *round,
	}); err != nil {
		logging.LogErrorWithMetadata(ctx, s.logger, msg, "Failed to publish round.entity.created event", map[string]interface{}{
			"error": err,
		})
		return fmt.Errorf("failed to publish round.entity.created event: %w", err)
	}

	logging.LogInfoWithMetadata(ctx, s.logger, msg, "Round entity created", map[string]interface{}{"round_id": round.ID})
	return nil
}

// StoreRound stores the round entity in the database.
func (s *RoundService) StoreRound(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundEntityCreatedPayload](msg, s.logger)
	if err != nil {
		return s.publishRoundError(msg, &roundevents.RoundCreateRequestPayload{}, []error{fmt.Errorf("invalid payload: %w", err)})
	}

	if err := s.RoundDB.CreateRound(ctx, &eventPayload.Round); err != nil {
		return s.publishRoundError(msg, &roundevents.RoundCreateRequestPayload{}, []error{err})
	}

	// Publish "round.stored" event
	if err := s.publishEvent(msg, roundevents.RoundStored, eventPayload); err != nil {
		logging.LogErrorWithMetadata(ctx, s.logger, msg, "Failed to publish round.stored event", map[string]interface{}{
			"error": err,
		})
		return fmt.Errorf("failed to publish round.stored event: %w", err)
	}

	logging.LogInfoWithMetadata(ctx, s.logger, msg, "Round stored in database", map[string]interface{}{"round_id": eventPayload.Round.ID})
	return nil
}

// PublishRoundCreated publishes the final round.created event.
func (s *RoundService) PublishRoundCreated(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundScheduledPayload](msg, s.logger)
	if err != nil {
		return s.publishRoundError(msg, &roundevents.RoundCreateRequestPayload{}, []error{fmt.Errorf("invalid payload: %w", err)})
	}

	round, err := s.RoundDB.GetRound(ctx, eventPayload.RoundID)
	if err != nil {
		return s.publishRoundError(msg, &roundevents.RoundCreateRequestPayload{}, []error{err})
	}

	// Publish the "round.created" event
	if err := s.publishEvent(msg, roundevents.RoundCreated, roundevents.RoundCreatedPayload{
		RoundID:   round.ID,
		Name:      round.Title,
		StartTime: round.StartTime,
	}); err != nil {
		logging.LogErrorWithMetadata(ctx, s.logger, msg, "Failed to publish round.created event", map[string]interface{}{
			"error": err,
		})
		return fmt.Errorf("failed to publish round.created event: %w", err)
	}

	logging.LogInfoWithMetadata(ctx, s.logger, msg, "Published round.created event", map[string]interface{}{"round_id": round.ID})
	return nil
}

// publishRoundError publishes a round.error event with details.
func (s *RoundService) publishRoundError(msg *message.Message, input *roundevents.RoundCreateRequestPayload, errs []error) error {
	// Format the errors into a string
	var errMsgBuilder strings.Builder
	for _, err := range errs {
		errMsgBuilder.WriteString(err.Error())
		errMsgBuilder.WriteString("; ")
	}
	errMsg := errMsgBuilder.String()
	if len(errMsg) > 2 {
		errMsg = errMsg[:len(errMsg)-2] // Remove trailing semicolon and space
	}

	payload := roundevents.RoundErrorPayload{
		CorrelationID: middleware.MessageCorrelationID(msg),
		Round:         input,
		Error:         errMsg, // Set the formatted error string
	}

	// Set caused_by metadata to the name of the calling function.
	if pubErr := s.publishEvent(msg, roundevents.RoundError, payload); pubErr != nil {
		logging.LogErrorWithMetadata(context.Background(), s.logger, msg, "Failed to publish round.error event", map[string]interface{}{
			"errors": errs,
		})
		return fmt.Errorf("failed to publish round.error event: %w, original errors: %v", pubErr, errs)
	}

	return fmt.Errorf("%s", errMsg) // Return the combined error message
}

// createRoundEntity creates a Round entity from the request payload.
func createRoundEntity(payload roundevents.RoundDateTimeParsedPayload) *roundtypes.Round {
	roundID := watermill.NewUUID()
	return &roundtypes.Round{
		ID:           roundID,
		Title:        payload.RoundCreateRequestPayload.Title,
		Location:     payload.RoundCreateRequestPayload.Location,
		EventType:    payload.RoundCreateRequestPayload.EventType,
		StartTime:    payload.StartTime,
		State:        roundtypes.RoundStateUpcoming,
		Participants: transformParticipants(payload.RoundCreateRequestPayload.Participants),
	}
}

// transformParticipants transforms a slice of ParticipantInput to a slice of RoundParticipant.
func transformParticipants(participants []roundtypes.ParticipantInput) []roundtypes.RoundParticipant {
	var transformed []roundtypes.RoundParticipant
	for _, p := range participants {
		tagNumber := 0
		if p.TagNumber != nil {
			tagNumber = *p.TagNumber
		}
		transformed = append(transformed, roundtypes.RoundParticipant{
			DiscordID: p.DiscordID,
			TagNumber: tagNumber,
			Response:  p.Response,
		})
	}
	return transformed
}
