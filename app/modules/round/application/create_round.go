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
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundCreateRequestPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	input := roundtypes.CreateRoundInput{
		Title:     eventPayload.Title,
		Location:  eventPayload.Location,
		EventType: eventPayload.EventType,
		StartTime: roundtypes.RoundTimeInput{
			Date: eventPayload.DateTime.Date,
			Time: eventPayload.DateTime.Time,
		},
	}

	if errs := s.roundValidator.ValidateRoundInput(input); len(errs) > 0 {
		return fmt.Errorf("validation errors: %v", errs)
	}

	// Publish "round.validated" event
	return s.publishEvent(msg, roundevents.RoundValidated, roundevents.RoundValidatedPayload{
		RoundCreateRequestPayload: eventPayload,
	})
}

// ParseDateTime parses and validates the event start time.
func (s *RoundService) ParseDateTime(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundValidatedPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	startTime, err := roundutil.NewDateTimeParser().ParseDateTime(
		eventPayload.RoundCreateRequestPayload.DateTime.Date + " " + eventPayload.RoundCreateRequestPayload.DateTime.Time)
	if err != nil {
		return fmt.Errorf("failed to parse date/time: %w", err)
	}

	// Publish "round.datetime.parsed" event
	return s.publishEvent(msg, roundevents.RoundDateTimeParsed, roundevents.RoundDateTimeParsedPayload{
		RoundCreateRequestPayload: eventPayload.RoundCreateRequestPayload,
		StartTime:                 startTime,
	})
}

// StoreRound saves the round in the database, ensuring no duplicates.
func (s *RoundService) StoreRound(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundEntityCreatedPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	// Prevent duplicate rounds
	if _, err := s.RoundDB.GetRound(ctx, eventPayload.Round.ID); err == nil {
		return fmt.Errorf("round already exists")
	} else if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to check for existing round: %w", err)
	}

	round := &roundtypes.Round{
		ID:        eventPayload.Round.ID,
		Title:     eventPayload.Round.Title,
		Location:  eventPayload.Round.Location,
		EventType: eventPayload.Round.EventType,
		StartTime: eventPayload.Round.StartTime,
	}

	if err := s.RoundDB.CreateRound(ctx, round); err != nil {
		return fmt.Errorf("failed to store round in database: %w", err)
	}

	// Publish "round.stored" event
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
