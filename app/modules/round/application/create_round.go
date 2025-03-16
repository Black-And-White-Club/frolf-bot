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
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/utils"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
)

// ValidateRoundRequest validates the round creation request.
func (s *RoundService) ValidateRoundRequest(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.CreateRoundRequestedPayload](msg, slog.Default())
	if err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	// Ensure StartTime is present
	if eventPayload.StartTime == "" {
		return fmt.Errorf("missing required field: start_time")
	}

	input := roundtypes.CreateRoundInput{ // Use roundtypes.CreateRoundInput
		Title:       eventPayload.Title,
		Description: &eventPayload.Description,
		Location:    &eventPayload.Location,
		StartTime:   eventPayload.StartTime,
		UserID:      eventPayload.UserID,
	}

	slog.Debug(input.StartTime)
	errs := s.roundValidator.ValidateRoundInput(input) // This should now work
	if len(errs) > 0 {
		slog.Error("Validation failed", "errors", errs)

		// Publish a failure event
		if publishErr := s.publishEvent(msg, roundevents.RoundValidationFailed, roundevents.RoundValidationFailedPayload{
			UserID:       roundtypes.UserID(eventPayload.UserID), // Convert string to roundtypes.UserID
			ErrorMessage: errs,
		}); publishErr != nil {
			slog.Error("Failed to publish validation failed event", "error", publishErr)
		}
		return nil
	}

	slog.Info("Publishing from ValidateRoundRequest Service Function", "payload", eventPayload)
	return s.publishEvent(msg, roundevents.RoundValidated, roundevents.RoundValidatedPayload{
		CreateRoundRequestedPayload: eventPayload,
	})
}

// ProcessValidatedRound transforms RoundValidatedPayload to RoundEntityCreatedPayload
func (s *RoundService) ProcessValidatedRound(ctx context.Context, msg *message.Message, timeParser roundutil.TimeParserInterface) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundValidatedPayload](msg, slog.Default())
	if err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	// Extract timezone from the event payload
	timezoneStr := eventPayload.CreateRoundRequestedPayload.Timezone

	clock := roundutil.RealClock{}

	// Parse StartTime string using 'when' and the submission timestamp, considering the user's time zone
	parsedTimeUnix, err := timeParser.ParseUserTimeInput(eventPayload.CreateRoundRequestedPayload.StartTime, timezoneStr, clock)
	if err != nil {
		// Publish a validation failed event if parsing fails
		if publishErr := s.publishEvent(msg, roundevents.RoundValidationFailed, roundevents.RoundValidationFailedPayload{
			UserID:       roundtypes.UserID(eventPayload.CreateRoundRequestedPayload.UserID),
			ErrorMessage: []string{err.Error()},
		}); publishErr != nil {
			slog.Error("Failed to publish validation failed event", "error", publishErr)
		}
		return nil
	}

	// Continue with the rest of the logic if parsing is successful
	parsedTime := time.Unix(parsedTimeUnix, 0).UTC() // Convert to UTC

	slog.Info("🟢 Time after parsing", "parsedTime", parsedTime.Format(time.RFC3339))

	// Create round entity
	roundTypes := roundtypes.Round{
		Title:        roundtypes.Title(eventPayload.CreateRoundRequestedPayload.Title),
		Description:  roundtypes.DescriptionPtr(eventPayload.CreateRoundRequestedPayload.Description),
		Location:     roundtypes.LocationPtr(eventPayload.CreateRoundRequestedPayload.Location),
		StartTime:    (*roundtypes.StartTime)(&parsedTime),
		CreatedBy:    roundtypes.UserID(eventPayload.CreateRoundRequestedPayload.UserID),
		State:        roundtypes.RoundStateUpcoming,
		Participants: []roundtypes.Participant{},
	}
	// Create event payload
	entityCreatedPayload := roundevents.RoundEntityCreatedPayload{
		Round:            roundTypes,
		DiscordChannelID: eventPayload.CreateRoundRequestedPayload.ChannelID,
		DiscordGuildID:   "",
	}

	slog.Info("Publishing from ProcessValidatedRound", "payload", entityCreatedPayload)
	return s.publishEvent(msg, roundevents.RoundEntityCreated, entityCreatedPayload)
}

func (s *RoundService) StoreRound(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundEntityCreatedPayload](msg, s.logger)
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

	s.logger.Info("About to CreateRound in DB", "payload", eventPayload)
	msg.Metadata.Set("user_id", string(roundDB.CreatedBy))

	// **Insert into the database**
	if err := s.RoundDB.CreateRound(ctx, &roundDB); err != nil {
		// Publish RoundCreationFailed event
		return s.publishEvent(msg, roundevents.RoundCreationFailed, roundevents.RoundCreationFailedPayload{
			UserID:       eventPayload.Round.CreatedBy,
			ErrorMessage: err.Error(),
		})
	}

	s.logger.Info("Round successfully stored in DB")

	// **Update roundTypes with the database ID**
	roundTypes.ID = roundDB.ID

	s.logger.Info("Round created successfully", "round_id", roundDB.ID)

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
