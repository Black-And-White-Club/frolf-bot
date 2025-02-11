package roundservice

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot/app/modules/round/domain/types"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/utils"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
)

// ValidateRoundRequest validates the round creation request.
func (s *RoundService) ValidateRoundRequest(ctx context.Context, msg *message.Message) error {
	// 1. Unmarshal the payload
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundCreateRequestPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	// 2. Create the CreateRoundInput object
	input := roundtypes.CreateRoundInput{
		Title:     eventPayload.Title,
		Location:  eventPayload.Location,
		EventType: eventPayload.EventType,
		StartTime: roundtypes.RoundTimeInput{
			Date: eventPayload.DateTime.Date,
			Time: eventPayload.DateTime.Time,
		},
		EndTime: roundtypes.RoundTimeInput{
			Date: eventPayload.EndTime.Date,
			Time: eventPayload.EndTime.Time,
		},
	}

	// 3. Validate the input
	if errs := s.roundValidator.ValidateRoundInput(input); len(errs) > 0 {
		return fmt.Errorf("validation errors: %v", errs)
	}

	// 4. Publish "round.validated" event
	return s.publishEvent(msg, roundevents.RoundValidated, roundevents.RoundValidatedPayload{
		RoundCreateRequestPayload: eventPayload,
	})
}

// ParseDateTime parses and validates the event start time and end time.
func (s *RoundService) ParseDateTime(ctx context.Context, msg *message.Message) error {
	// 1. Unmarshal the payload
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundValidatedPayload](msg, s.logger)
	if err != nil {
		s.logger.Error("Failed to unmarshal payload", "error", err)
		return fmt.Errorf("invalid payload: %w", err)
	}

	// Ensure eventPayload is not nil before accessing its fields
	if eventPayload.RoundCreateRequestPayload.Title == "" && eventPayload.RoundCreateRequestPayload.Location == "" {
		s.logger.Error("UnmarshalPayload returned empty eventPayload")
		return fmt.Errorf("unexpected empty payload")
	}

	// 2. Parse StartTime
	startTime, err := roundutil.NewDateTimeParser().ParseDateTime(
		eventPayload.RoundCreateRequestPayload.DateTime.Date + " " + eventPayload.RoundCreateRequestPayload.DateTime.Time)
	if err != nil {
		s.logger.Error("Failed to parse start date/time", "error", err)
		return fmt.Errorf("failed to parse start date/time: %w", err)
	}

	// 3. Parse EndTime
	endTime, err := roundutil.NewDateTimeParser().ParseDateTime(
		eventPayload.RoundCreateRequestPayload.EndTime.Date + " " + eventPayload.RoundCreateRequestPayload.EndTime.Time)
	if err != nil {
		s.logger.Error("Failed to parse end date/time", "error", err)
		return fmt.Errorf("failed to parse end date/time: %w", err)
	}

	// 4. Publish "round.datetime.parsed" event
	s.logger.Info("Publishing round.datetime.parsed event")
	return s.publishEvent(msg, roundevents.RoundDateTimeParsed, roundevents.RoundDateTimeParsedPayload{
		RoundCreateRequestPayload: eventPayload.RoundCreateRequestPayload,
		StartTime:                 &startTime,
		EndTime:                   &endTime,
	})
}

// StoreRound saves the round in the database.
func (s *RoundService) StoreRound(ctx context.Context, msg *message.Message) error {
	// 1. Unmarshal the payload
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundEntityCreatedPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	// 2. Create the Round object
	round := &roundtypes.Round{
		ID:             eventPayload.Round.ID,
		Title:          eventPayload.Round.Title,
		Location:       eventPayload.Round.Location,
		EventType:      eventPayload.Round.EventType,
		StartTime:      eventPayload.Round.StartTime,
		EndTime:        eventPayload.Round.EndTime,
		Finalized:      eventPayload.Round.Finalized,
		CreatedBy:      eventPayload.Round.CreatedBy,
		State:          eventPayload.Round.State,
		Participants:   eventPayload.Round.Participants,
		DiscordEventID: eventPayload.Round.DiscordEventID,
	}

	// Prevent duplicate rounds
	if _, err := s.RoundDB.GetRound(ctx, eventPayload.Round.ID); err == nil {
		return fmt.Errorf("round already exists")
	} else if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to check for existing round: %w", err)
	}

	// 3. Store the round in the database
	if err := s.RoundDB.CreateRound(ctx, round); err != nil {
		return fmt.Errorf("failed to store round in database: %w", err)
	}

	// 4. Publish "round.stored" event
	return s.publishEvent(msg, roundevents.RoundStored, eventPayload)
}

// PublishRoundCreated publishes the final event after storage.
func (s *RoundService) PublishRoundCreated(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundScheduledPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	round, err := s.RoundDB.GetRound(ctx, eventPayload.RoundID)
	if err != nil {
		return fmt.Errorf("failed to get round from database: %w", err)
	}

	// Publish "round.created" event
	return s.publishEvent(msg, roundevents.RoundCreated, round)
}

// UpdateDiscordEventID updates the DiscordEventID for an existing round.
func (s *RoundService) UpdateDiscordEventID(ctx context.Context, msg *message.Message) error {
	// 1. Unmarshal the payload (this will likely be a new payload type)
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundEventCreatedPayload](msg, s.logger) // Use the correct payload type
	if err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	// 2. Update the DiscordEventID in the database
	if err := s.RoundDB.UpdateDiscordEventID(ctx, eventPayload.RoundID, eventPayload.DiscordEventID); err != nil {
		return fmt.Errorf("failed to update Discord event ID: %w", err)
	}

	// 3. Publish "round.discord_event_id.updated" event
	if err := s.publishEvent(msg, roundevents.RoundDiscordEventIDUpdated, roundevents.RoundDiscordEventIDUpdatedPayload{
		RoundID:        eventPayload.RoundID,
		DiscordEventID: eventPayload.DiscordEventID,
	}); err != nil {
		return fmt.Errorf("failed to publish round discord event ID updated event: %w", err)
	}

	return nil
}
