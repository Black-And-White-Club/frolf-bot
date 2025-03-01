package roundservice

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
)

// ValidateRoundRequest validates the round creation request.
func (s *RoundService) ValidateRoundRequest(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundCreateRequestPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	// Ensure StartTime is present
	if eventPayload.StartTime == nil {
		return fmt.Errorf("missing required field: start_time")
	}

	input := roundtypes.CreateRoundInput{
		Title:       eventPayload.Title,
		Description: eventPayload.Description,
		Location:    eventPayload.Location,
		StartTime:   eventPayload.StartTime,
	}

	if errs := s.roundValidator.ValidateRoundInput(input); len(errs) > 0 {
		s.logger.Error("Validation failed", "errors", errs)

		// Publish a failure event
		return s.publishEvent(msg, roundevents.RoundValidationFailed, roundevents.RoundValidationFailedPayload{
			UserID:       eventPayload.UserID,
			ErrorMessage: &errs,
		})
	}

	s.logger.Info("Publishing RoundValidated event", "payload", eventPayload)
	return s.publishEvent(msg, roundevents.RoundValidated, roundevents.RoundValidatedPayload{
		RoundCreateRequestPayload: eventPayload,
	})
}

// StoreRound saves the round in the database.
func (s *RoundService) StoreRound(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundEntityCreatedPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	round := eventPayload.Round

	// Prevent duplicate rounds
	existingRound, err := s.RoundDB.GetRound(ctx, round.ID)
	if err == nil {
		return fmt.Errorf("round already exists: %+v", existingRound)
	} else if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to check for existing round: %w", err)
	}

	if err := s.RoundDB.CreateRound(ctx, &round); err != nil {
		return fmt.Errorf("failed to store round in database: %w", err)
	}

	s.logger.Info("Publishing RoundStored event")
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

	s.logger.Info("Publishing RoundCreated event for Discord integration")
	return s.publishEvent(msg, roundevents.RoundCreated, round)
}

// UpdateDiscordEventID updates the DiscordEventID for an existing round.
func (s *RoundService) UpdateDiscordEventID(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundEventCreatedPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	if eventPayload.RoundID == "" {
		return fmt.Errorf("invalid RoundID in payload")
	}

	err = s.RoundDB.UpdateDiscordEventID(ctx, eventPayload.RoundID, eventPayload.DiscordEventID)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("round not found: %w", err)
	} else if err != nil {
		return fmt.Errorf("failed to update Discord event ID: %w", err)
	}

	s.logger.Info("Publishing RoundDiscordEventIDUpdated event")
	return s.publishEvent(msg, roundevents.RoundDiscordEventIDUpdated, roundevents.RoundDiscordEventIDUpdatedPayload(eventPayload))
}
