package roundhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// ReminderHandler handles reminder events.
type ReminderHandler struct {
	RoundDB  rounddb.RoundDB
	PubSuber watermillutil.PubSuber
}

// NewReminderHandler creates a new ReminderHandler.
func NewReminderHandler(roundDB rounddb.RoundDB, pubsuber watermillutil.PubSuber) *ReminderHandler {
	return &ReminderHandler{
		RoundDB:  roundDB,
		PubSuber: pubsuber,
	}
}

// Handle processes reminder events.
func (h *ReminderHandler) Handle(ctx context.Context, msg *message.Message) error {
	var event RoundReminderEvent
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		return fmt.Errorf("failed to unmarshal ReminderEvent: %w", err)
	}

	// Fetch the round from the database using event.RoundID
	round, err := h.RoundDB.GetRound(ctx, event.RoundID)
	if err != nil {
		return fmt.Errorf("failed to get round: %w", err)
	}

	// Check if the round still exists and is in the upcoming state
	if round == nil || round.State != rounddb.RoundStateUpcoming {
		return nil // Ignore the reminder if the round doesn't exist or is not upcoming
	}

	// Calculate reminder time based on round start time and reminder type
	reminderTime := calculateReminderTime(round.Date, round.Time, event.ReminderType)

	// Check if the reminder time is in the past
	if time.Now().After(reminderTime) {
		// Construct the reminder payload
		payload := map[string]interface{}{
			"round_id":      round.ID,
			"reminder_type": event.ReminderType,
			// Add other relevant data here, like round title, date, time, etc.
		}

		// Marshal the payload into JSON
		jsonPayload, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal reminder payload: %w", err)
		}

		// Publish the payload directly using the topic
		if err := h.PubSuber.Publish(TopicRoundReminder, message.NewMessage(watermill.NewUUID(), jsonPayload)); err != nil {
			return fmt.Errorf("failed to publish reminder event: %w", err)
		}
	}

	return nil
}

// Helper function to calculate the reminder time
func calculateReminderTime(roundDate time.Time, roundTime string, reminderType string) time.Time {
	// Parse the roundTime string into a time.Time
	startTime, err := time.Parse("15:04", roundTime)
	if err != nil {
		// Handle the error appropriately
		return time.Time{}
	}

	// Combine the roundDate and startTime to get the complete start time
	roundStartTime := time.Date(
		roundDate.Year(), roundDate.Month(), roundDate.Day(),
		startTime.Hour(), startTime.Minute(), 0, 0,
		roundDate.Location(),
	)

	switch reminderType {
	case "one-hour":
		return roundStartTime.Add(-1 * time.Hour)
	case "thirty-minutes":
		return roundStartTime.Add(-30 * time.Minute)
	default:
		// Handle invalid reminder type
		return roundStartTime
	}
}
