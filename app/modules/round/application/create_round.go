package roundservice

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/utils"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
)

// ValidateRoundRequest validates the round creation request.
func (s *RoundService) ValidateRoundRequest(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundCreateRequestPayload](msg, slog.Default())
	if err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	// Ensure StartTime is present
	if eventPayload.StartTime == "" {
		return fmt.Errorf("missing required field: start_time")
	}

	input := roundtypes.CreateRoundInput{
		Title:       eventPayload.Title,
		Description: eventPayload.Description,
		Location:    eventPayload.Location,
		StartTime:   eventPayload.StartTime,
	}

	slog.Debug(input.StartTime)
	errs := s.roundValidator.ValidateRoundInput(input)
	if len(errs) > 0 {
		slog.Error("Validation failed", "errors", errs)

		// Publish a failure event
		if publishErr := s.publishEvent(msg, roundevents.RoundValidationFailed, roundevents.RoundValidationFailedPayload{
			UserID:       eventPayload.UserID,
			ErrorMessage: errs,
		}); publishErr != nil {
			slog.Error("Failed to publish validation failed event", "error", publishErr)
		}
		return nil // Ensure you return here to stop further processing
	}

	// Update eventPayload.StartTime with the parsed time
	eventPayload.StartTime = input.StartTime

	slog.Info("Publishing from ValidateRoundRequest Service Function", "payload", eventPayload)
	return s.publishEvent(msg, roundevents.RoundValidated, roundevents.RoundValidatedPayload{
		RoundCreateRequestPayload: eventPayload,
	})
}

// ProcessValidatedRound transforms RoundValidatedPayload to RoundEntityCreatedPayload
func (s *RoundService) ProcessValidatedRound(ctx context.Context, msg *message.Message, timeParser roundutil.TimeParserInterface) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundValidatedPayload](msg, slog.Default())
	if err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	// Extract timezone from the event payload
	timezoneStr := eventPayload.RoundCreateRequestPayload.Timezone

	clock := roundutil.RealClock{}

	// Parse StartTime string using 'when' and the submission timestamp, considering the user's time zone
	parsedTimeUnix, err := timeParser.ParseUserTimeInput(eventPayload.RoundCreateRequestPayload.StartTime, timezoneStr, clock)
	if err != nil {
		// Publish a validation failed event if parsing fails
		if publishErr := s.publishEvent(msg, roundevents.RoundValidationFailed, roundevents.RoundValidationFailedPayload{
			UserID:       eventPayload.RoundCreateRequestPayload.UserID,
			ErrorMessage: []string{err.Error()},
		}); publishErr != nil {
			slog.Error("Failed to publish validation failed event", "error", publishErr)
		}
		return nil // Return nil to indicate that the event was published
	}

	// Continue with the rest of the logic if parsing is successful
	parsedTime := time.Unix(parsedTimeUnix, 0).UTC() // Convert to UTC

	// Create round entity
	roundTypes := roundtypes.Round{
		Title:        eventPayload.RoundCreateRequestPayload.Title,
		Description:  eventPayload.RoundCreateRequestPayload.Description,
		Location:     eventPayload.RoundCreateRequestPayload.Location,
		StartTime:    &parsedTime,
		CreatedBy:    eventPayload.RoundCreateRequestPayload.UserID,
		State:        roundtypes.RoundStateUpcoming,
		Participants: []roundtypes.RoundParticipant{},
	}
	slog.Info("Start Time right now dereferenced", &parsedTime)
	slog.Info("Start Time right now normal", parsedTime)

	// Create event payload
	entityCreatedPayload := roundevents.RoundEntityCreatedPayload{
		Round:            roundTypes,
		DiscordChannelID: eventPayload.RoundCreateRequestPayload.ChannelID,
		DiscordGuildID:   "", // This field is still empty, consider populating it with data
	}

	slog.Info("Publishing from ProcessValidatedRound", "payload", entityCreatedPayload)
	// Publish RoundEntityCreated event
	return s.publishEvent(msg, roundevents.RoundEntityCreated, entityCreatedPayload)
}

// StoreRound stores a round in the database.
func (s *RoundService) StoreRound(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundEntityCreatedPayload](msg, slog.Default())
	if err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	roundTypes := eventPayload.Round // Extract the round object

	// Ensure required fields are present
	if roundTypes.Description == nil {
		return fmt.Errorf("description is required but was nil")
	}
	if roundTypes.Location == nil {
		return fmt.Errorf("location is required but was nil")
	}
	if roundTypes.StartTime == nil {
		return fmt.Errorf("startTime is required but was nil")
	}

	// **Map round data to the database model**
	roundDB := rounddb.Round{
		Title:       roundTypes.Title,
		Description: *roundTypes.Description,
		Location:    *roundTypes.Location,
		EventType:   roundTypes.EventType,
		StartTime:   *roundTypes.StartTime,
		Finalized:   roundTypes.Finalized,
		CreatorID:   roundTypes.CreatedBy,
		State:       rounddb.RoundState(roundTypes.State),
	}

	slog.Info("About to CreateRound in DB", "payload", eventPayload)
	msg.Metadata.Set("user_id", roundDB.CreatorID)
	// **Insert into the database**
	if err := s.RoundDB.CreateRound(ctx, &roundDB); err != nil {
		// Publish RoundCreationFailed event
		return s.publishEvent(msg, roundevents.RoundCreationFailed, roundevents.RoundCreationFailedPayload{
			UserID:       eventPayload.Round.CreatedBy,
			ErrorMessage: err.Error(),
		})
	}

	slog.Info("Round successfully stored in DB")

	// **Update roundTypes with the database ID**
	roundTypes.ID = roundDB.ID

	slog.Info("Round created successfully", "round_id", roundDB.ID)

	// Publish RoundCreated event
	return s.publishEvent(msg, roundevents.RoundStored, roundevents.RoundStoredPayload{
		Round: roundTypes,
	})
}

// PublishRoundCreated publishes the final event after storage.
func (s *RoundService) PublishRoundCreated(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundScheduledPayload](msg, slog.Default())
	if err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	round := roundevents.RoundCreatedPayload{
		RoundID:     eventPayload.RoundID,
		Title:       eventPayload.Title,
		Description: eventPayload.Description,
		StartTime:   eventPayload.StartTime,
		Location:    eventPayload.Location,
		CreatedBy:   eventPayload.CreatedBy,
	}

	slog.Info("Publishing RoundCreated event for Discord integration")
	return s.publishEvent(msg, roundevents.RoundCreated, round)
}

// UpdateDiscordEventID updates the DiscordEventID for an existing round.
func (s *RoundService) UpdateDiscordEventID(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundEventCreatedPayload](msg, slog.Default())
	if err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	if eventPayload.RoundID == 0 { // Check if RoundID is zero
		return fmt.Errorf("invalid RoundID in payload")
	}

	err = s.RoundDB.UpdateDiscordEventID(ctx, eventPayload.RoundID, eventPayload.DiscordEventID)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("round not found: %w", err)
	} else if err != nil {
		return fmt.Errorf("failed to update Discord event ID: %w", err)
	}

	slog.Info("Publishing RoundDiscordEventIDUpdated event")
	return s.publishEvent(msg, roundevents.RoundDiscordEventIDUpdated, roundevents.RoundDiscordEventIDUpdatedPayload(eventPayload))
}
