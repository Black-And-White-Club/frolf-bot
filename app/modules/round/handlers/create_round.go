package roundhandlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	roundcommands "github.com/Black-And-White-Club/tcr-bot/app/modules/round/commands"
	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
	"github.com/Black-And-White-Club/tcr-bot/internal/jetstream"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// CreateRoundHandler handles the CreateRoundRequest command.
type CreateRoundHandler struct {
	roundDB    rounddb.RoundDB
	messageBus watermillutil.Publisher
}

// NewCreateRoundHandler creates a new CreateRoundHandler.
func NewCreateRoundHandler(roundDB rounddb.RoundDB, messageBus watermillutil.Publisher) *CreateRoundHandler {
	return &CreateRoundHandler{
		roundDB:    roundDB,
		messageBus: messageBus,
	}
}

// Handler processes the CreateRoundRequest command.
func (h *CreateRoundHandler) Handler(msg *message.Message) error {
	var cmd roundcommands.CreateRoundRequest
	if err := json.Unmarshal(msg.Payload, &cmd); err != nil {
		return fmt.Errorf("failed to unmarshal CreateRoundRequest: %w", err)
	}

	if cmd.Input.Title == "" {
		return errors.New("title is required")
	}

	round, err := h.roundDB.CreateRound(context.Background(), cmd.Input)
	if err != nil {
		return fmt.Errorf("failed to create round: %w", err)
	}

	// Publish a RoundCreated event
	event := RoundCreatedEvent{
		RoundID: round.ID,
		Input:   cmd.Input,
		// ... other relevant data from the round ...
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal RoundCreatedEvent: %w", err)
	}
	if err := h.messageBus.Publish(event.Topic(), message.NewMessage(watermill.NewUUID(), payload)); err != nil {
		return fmt.Errorf("failed to publish RoundCreatedEvent: %w", err)
	}

	// Schedule the round start event
	roundStartTime := calculateRoundStartTime(round.Date, round.Time)
	if err := jetstream.PublishScheduledMessage(
		context.Background(),
		h.messageBus.(*watermillutil.PubSub), // Pass the PubSub instance directly
		"scheduled_tasks",
		round.ID,
		"StartRoundEventHandler",
		roundStartTime,
	); err != nil {
		return fmt.Errorf("failed to schedule round start event: %w", err)
	}

	// Schedule the reminder events (1 hour and 30 minutes before)
	if err := jetstream.PublishScheduledMessage(
		context.Background(),
		h.messageBus.(*watermillutil.PubSub),
		"scheduled_tasks",
		round.ID,
		"ReminderOneHourHandler",
		roundStartTime.Add(-1*time.Hour),
	); err != nil {
		return fmt.Errorf("failed to schedule one-hour reminder: %w", err)
	}

	if err := jetstream.PublishScheduledMessage(
		context.Background(),
		h.messageBus.(*watermillutil.PubSub),
		"scheduled_tasks",
		round.ID,
		"ReminderThirtyMinutesHandler",
		roundStartTime.Add(-30*time.Minute),
	); err != nil {
		return fmt.Errorf("failed to schedule thirty-minutes reminder: %w", err)
	}

	return nil
}

// Helper function to calculate the round start time (similar to calculateReminderTime)
func calculateRoundStartTime(roundDate time.Time, roundTime string) time.Time {
	// Parse the roundTime string into a time.Time
	startTime, err := time.Parse("15:04", roundTime) // Assuming your roundTime is in "HH:MM" format
	if err != nil {
		// Handle the error appropriately (e.g., log an error and return a default value)
		return time.Time{}
	}

	// Combine the roundDate and startTime to get the complete start time
	return time.Date(
		roundDate.Year(), roundDate.Month(), roundDate.Day(),
		startTime.Hour(), startTime.Minute(), 0, 0,
		roundDate.Location(),
	)
}
