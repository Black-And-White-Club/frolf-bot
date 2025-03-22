package roundservice

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	roundtime "github.com/Black-And-White-Club/frolf-bot/app/modules/round/time_utils"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/utils"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
)

// ValidateRoundRequest validates the round creation request
func (s *RoundService) ValidateRoundRequest(ctx context.Context, payload roundevents.CreateRoundRequestedPayload) error {
	// Ensure StartTime is present
	if payload.StartTime == "" {
		return fmt.Errorf("missing required field: start_time")
	}

	input := roundtypes.CreateRoundInput{
		Title:       payload.Title,
		Description: &payload.Description,
		Location:    &payload.Location,
		StartTime:   payload.StartTime,
		UserID:      payload.UserID,
	}

	s.logger.Debug("Validating round input", "startTime", attr.Time(input.StartTime))
	errs := s.roundValidator.ValidateRoundInput(input)
	if len(errs) > 0 {
		s.logger.Error("Validation failed", attr.Error(errs))

		// Publish a failure event
		failedPayload := roundevents.RoundValidationFailedPayload{
			UserID:       roundtypes.UserID(payload.UserID),
			ErrorMessage: errs,
		}

		if err := s.publishEvent(roundevents.RoundValidationFailed, failedPayload); err != nil {
			s.logger.Error("Failed to publish validation failed event", "error", err)
		}
		return fmt.Errorf("validation failed: %v", errs)
	}

	s.logger.Info("Round validation successful", "payload", payload)

	// Publish the validated payload
	validatedPayload := roundevents.RoundValidatedPayload{
		CreateRoundRequestedPayload: payload,
	}

	return s.publishEvent(roundevents.RoundValidated, validatedPayload)
}

// ProcessValidatedRound transforms validated round data to an entity
func (s *RoundService) ProcessValidatedRound(ctx context.Context, payload roundevents.RoundValidatedPayload, timeParser roundtime.TimeParserInterface) error {
	clock := roundutil.RealClock{}

	// Parse StartTime
	parsedTimeUnix, err := timeParser.ParseUserTimeInput(
		payload.CreateRoundRequestedPayload.StartTime,
		payload.CreateRoundRequestedPayload.Timezone,
		clock,
	)
	if err != nil {
		// Handle parsing failure
		failurePayload := roundevents.RoundValidationFailedPayload{
			UserID:       roundtypes.UserID(payload.CreateRoundRequestedPayload.UserID),
			ErrorMessage: []string{err.Error()},
		}

		if publishErr := s.publishEvent(roundevents.RoundValidationFailed, failurePayload); publishErr != nil {
			s.logger.Error("Failed to publish validation failed event", "error", publishErr)
		}
		return fmt.Errorf("time parsing failed: %w", err)
	}

	// Continue with the rest of the logic if parsing is successful
	parsedTime := time.Unix(parsedTimeUnix, 0).UTC() // Convert to UTC

	s.logger.Info("Time after parsing", "parsedTime", parsedTime.Format(time.RFC3339))

	// Create round entity
	roundTypes := roundtypes.Round{
		Title:        roundtypes.Title(payload.CreateRoundRequestedPayload.Title),
		Description:  &payload.CreateRoundRequestedPayload.Description,
		Location:     &payload.CreateRoundRequestedPayload.Location,
		StartTime:    (*roundtypes.StartTime)(&parsedTime),
		CreatedBy:    roundtypes.UserID(payload.CreateRoundRequestedPayload.UserID),
		State:        roundtypes.RoundStateUpcoming,
		Participants: []roundtypes.Participant{},
	}

	// Create event payload
	entityCreatedPayload := roundevents.RoundEntityCreatedPayload{
		Round:            roundTypes,
		DiscordChannelID: payload.CreateRoundRequestedPayload.ChannelID,
		DiscordGuildID:   "",
	}

	s.logger.Info("Entity created for round", "payload", entityCreatedPayload)
	return s.publishEvent(roundevents.RoundEntityCreated, entityCreatedPayload)
}

// StoreRound stores a round in the database
func (s *RoundService) StoreRound(ctx context.Context, payload roundevents.RoundEntityCreatedPayload) error {
	roundTypes := payload.Round

	// Validate required fields
	if roundTypes.Description == nil {
		return fmt.Errorf("description is required but was nil")
	}
	if roundTypes.Location == nil {
		return fmt.Errorf("location is required but was nil")
	}
	if roundTypes.StartTime == nil {
		return fmt.Errorf("startTime is required but was nil")
	}

	// Map round data to the database model
	roundDB := roundtypes.Round{
		Title:       roundTypes.Title,
		Description: roundTypes.Description,
		Location:    roundTypes.Location,
		EventType:   roundTypes.EventType,
		StartTime:   roundTypes.StartTime,
		Finalized:   roundTypes.Finalized,
		CreatedBy:   roundTypes.CreatedBy,
		State:       roundTypes.State,
	}

	s.logger.Info("About to create round in DB", "payload", payload)

	// Insert into the database
	if err := s.RoundDB.CreateRound(ctx, &roundDB); err != nil {
		// Publish RoundCreationFailed event
		failurePayload := roundevents.RoundCreationFailedPayload{
			UserID:       payload.Round.CreatedBy,
			ErrorMessage: err.Error(),
		}

		if publishErr := s.publishEvent(roundevents.RoundCreationFailed, failurePayload); publishErr != nil {
			s.logger.Error("Failed to publish creation failed event", "error", publishErr)
		}
		return fmt.Errorf("failed to create round: %w", err)
	}

	s.logger.Info("Round successfully stored in DB")

	// Update roundTypes with the database ID
	roundTypes.ID = roundDB.ID

	s.logger.Info("Round created successfully", "round_id", roundDB.ID)

	// Publish RoundStored event
	storedPayload := roundevents.RoundStoredPayload{
		Round: roundTypes,
	}

	return s.publishEvent(roundevents.RoundStored, storedPayload)
}

// PublishRoundCreated publishes the final event after storage.
func (s *RoundService) PublishRoundCreated(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundScheduledPayload](msg, slog.Default())
	if err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	round := roundevents.RoundCreatedPayload{
		BaseRoundPayload: roundtypes.BaseRoundPayload{
			RoundID:     eventPayload.RoundID,
			Title:       eventPayload.Title,
			Description: eventPayload.Description,
			Location:    eventPayload.Location,
			StartTime:   eventPayload.StartTime,
			UserID:      eventPayload.UserID,
		},
	}

	slog.Info("Publishing RoundCreated event for Discord integration")
	return s.publishEvent(msg, roundevents.RoundCreated, round)
}

// UpdateEventMessageID updates the EventMessageID for an existing round (MessageID so it can be used for interactions later)
func (s *RoundService) UpdateEventMessageID(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundEventMessageIDUpdatedPayload](msg, slog.Default())
	if err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	if eventPayload.RoundID == 0 { // Check if RoundID is zero
		return fmt.Errorf("invalid RoundID in payload")
	}

	err = s.RoundDB.UpdateEventMessageID(ctx, eventPayload.RoundID, eventPayload.EventMessageID)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("round not found: %w", err)
	} else if err != nil {
		return fmt.Errorf("failed to update Discord event ID: %w", err)
	}

	slog.Info("Publishing RoundEventMessageIDUpdated event")
	return s.publishEvent(msg, roundevents.RoundEventMessageIDUpdated, roundevents.RoundEventMessageIDUpdatedPayload(eventPayload))
}
