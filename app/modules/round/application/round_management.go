package roundservice

import (
	"context"
	"fmt"
	"time"

	roundevents "github.com/Black-And-White-Club/tcr-bot/app/modules/round/domain/events"
	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/infrastructure/repositories" // Importing the correct package
	roundutil "github.com/Black-And-White-Club/tcr-bot/app/modules/round/utils"
	"github.com/ThreeDotsLabs/watermill"
)

// CreateRound handles the RoundCreateRequest event.
func (s *RoundService) CreateRound(ctx context.Context, input roundevents.RoundCreateRequestPayload) error {
	// Generate a new UUID for the round
	roundID := watermill.NewUUID()

	// Validate input fields
	if input.Title == "" {
		return fmt.Errorf("title cannot be empty")
	}
	if input.DateTime.Date == "" || input.DateTime.Time == "" {
		return fmt.Errorf("date/time input cannot be empty")
	}

	// Create a DateTimeParser instance
	parser := roundutil.NewDateTimeParser()

	// Parse the date/time input using the parser
	startTime, err := parser.ParseDateTime(input.DateTime.Date + " " + input.DateTime.Time)
	if err != nil {
		// Publish an event indicating the parsing error
		errMsg := "Invalid date/time format. Please try a different format."
		err = s.publishEvent(ctx, roundevents.RoundCreateResponse, &roundevents.RoundCreateResponsePayload{
			Success: false,
			RoundID: "",
			Error:   errMsg,
		})
		if err != nil {
			return fmt.Errorf("failed to publish round creation failed event: %w", err)
		}
		return fmt.Errorf("failed to parse date/time: %w", err)
	}

	// Convert input to Round model
	round := &rounddb.Round{
		ID:           roundID,
		Title:        input.Title,
		Location:     input.Location,
		EventType:    input.EventType,
		Date:         startTime.Truncate(24 * time.Hour),
		Time:         startTime,
		State:        rounddb.RoundStateUpcoming,
		Participants: make([]rounddb.Participant, 0),
	}

	// Convert Participants from input to database model
	for _, p := range input.Participants {
		tagNumber := 0
		if p.TagNumber != nil {
			tagNumber = *p.TagNumber
		}
		round.Participants = append(round.Participants, rounddb.Participant{
			DiscordID: p.DiscordID,
			TagNumber: &tagNumber,
			Response:  rounddb.Response(p.Response),
		})
	}

	err = s.RoundDB.CreateRound(ctx, round)
	if err != nil {
		return fmt.Errorf("failed to create round: %w", err)
	}

	// Publish RoundCreatedEvent with the generated RoundID
	err = s.publishEvent(ctx, roundevents.RoundCreated, &roundevents.RoundCreatedPayload{
		RoundID:      roundID,
		Name:         input.Title,
		StartTime:    startTime,
		Participants: input.Participants,
	})
	if err != nil {
		return fmt.Errorf("failed to publish round created event: %w", err)
	}

	// Schedule reminders and round start
	err = s.scheduleRoundEvents(ctx, roundID, startTime)
	if err != nil {
		return fmt.Errorf("failed to schedule round events: %w", err)
	}

	return nil
}

// UpdateRound handles the RoundUpdatedEvent.
func (s *RoundService) UpdateRound(ctx context.Context, event *roundevents.RoundUpdatedPayload) error { // Using RoundUpdatedPayload
	// 1. Fetch the round from the database
	round, err := s.RoundDB.GetRound(ctx, event.RoundID)
	if err != nil {
		return fmt.Errorf("failed to get round: %w", err)
	}

	// 2. Update the round with new values
	if event.Title != nil {
		round.Title = *event.Title
	}
	if event.Location != nil {
		round.Location = *event.Location
	}
	if event.EventType != nil {
		round.EventType = event.EventType
	}
	if event.Date != nil {
		round.Date = *event.Date
	}
	if event.Time != nil {
		round.Time = *event.Time
	}

	// 3. Update the round in the database
	err = s.RoundDB.UpdateRound(ctx, event.RoundID, round) // Pass the updated Round object
	if err != nil {
		return fmt.Errorf("failed to update round: %w", err)
	}

	// 4. Publish a RoundUpdatedEvent with the updated values
	updatedEvent := &roundevents.RoundUpdatedPayload{ // Using RoundUpdatedPayload
		RoundID:   event.RoundID,
		Title:     &round.Title,
		Location:  &round.Location,
		EventType: round.EventType,
		Date:      &round.Date,
		Time:      &round.Time,
	}
	err = s.publishEvent(ctx, roundevents.RoundUpdated, updatedEvent) // Publish RoundUpdated
	if err != nil {
		return fmt.Errorf("failed to publish round updated event: %w", err)
	}

	return nil
}

// DeleteRound handles the RoundDeletedEvent.
func (s *RoundService) DeleteRound(ctx context.Context, event *roundevents.RoundDeletedPayload) error { // Using RoundDeletedPayload
	err := s.RoundDB.DeleteRound(ctx, event.RoundID)
	if err != nil {
		return fmt.Errorf("failed to delete round: %w", err)
	}

	// Publish a RoundDeletedEvent
	err = s.publishEvent(ctx, roundevents.RoundDeleted, &roundevents.RoundDeletedPayload{ // Publish RoundDeleted
		RoundID: event.RoundID,
		State:   rounddb.RoundStateDeleted,
	})
	if err != nil {
		return fmt.Errorf("failed to publish round deleted event: %w", err)
	}

	return nil
}

// StartRound handles the RoundStartedEvent (triggered by the scheduler).
func (s *RoundService) StartRound(ctx context.Context, event *roundevents.RoundStartedPayload) error { // Using RoundStartedPayload
	// 1. Fetch the round from the database
	round, err := s.RoundDB.GetRound(ctx, event.RoundID)
	if err != nil {
		return fmt.Errorf("failed to get round: %w", err)
	}

	// 2. Update the round state in the database
	err = s.RoundDB.UpdateRoundState(ctx, event.RoundID, rounddb.RoundStateInProgress)
	if err != nil {
		return fmt.Errorf("failed to update round state: %w", err)
	}

	// 3. Prepare a list of participants with accepted/tentative responses
	participants := make([]roundevents.Participant, 0)
	for _, p := range round.Participants {
		if p.Response == rounddb.ResponseAccept || p.Response == rounddb.ResponseTentative {
			var tagNumber int
			if p.TagNumber != nil {
				tagNumber = *p.TagNumber
			}
			participants = append(participants, roundevents.Participant{
				DiscordID: p.DiscordID,
				TagNumber: tagNumber,
			})
		}
	}

	// 4. Publish a RoundStartedEvent with the participants
	err = s.publishEvent(ctx, roundevents.RoundStarted, &roundevents.RoundStartedPayload{ // Publish RoundStarted
		RoundID:      event.RoundID,
		Participants: participants,
	})
	if err != nil {
		return fmt.Errorf("failed to publish round started event: %w", err)
	}

	return nil
}
