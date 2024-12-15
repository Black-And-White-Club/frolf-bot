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

// Define a custom type for the round ID key
type roundIDCtxKey string

// ScheduledTaskHandler handles scheduled task events.
type ScheduledTaskHandler struct {
	RoundDB  rounddb.RoundDB
	PubSuber watermillutil.PubSuber
}

// NewScheduledTaskHandler creates a new ScheduledTaskHandler.
func NewScheduledTaskHandler(roundDB rounddb.RoundDB, pubsuber watermillutil.PubSuber) *ScheduledTaskHandler {
	return &ScheduledTaskHandler{
		RoundDB:  roundDB,
		PubSuber: pubsuber,
	}
}

// Handle processes scheduled task events.
func (h *ScheduledTaskHandler) Handle(ctx context.Context, msg *message.Message) error {
	var taskData map[string]interface{}
	if err := json.Unmarshal(msg.Payload, &taskData); err != nil {
		return fmt.Errorf("failed to unmarshal task data: %w", err)
	}

	// Extract the scheduled time from the message header (if needed)
	scheduledAtStr := msg.Metadata.Get("scheduled_at")
	if scheduledAtStr != "" {
		scheduledAt, err := time.Parse(time.RFC3339, scheduledAtStr)
		if err != nil {
			return fmt.Errorf("failed to parse scheduled_at: %w", err)
		}

		// Check if the scheduled time is in the past
		if time.Now().Before(scheduledAt) {
			return nil // Not time to execute the task yet
		}
	}

	roundID, ok := taskData["round_id"].(float64)
	if !ok {
		return fmt.Errorf("invalid round_id")
	}

	handlerName, ok := taskData["handler"].(string)
	if !ok {
		return fmt.Errorf("invalid handler")
	}

	// Create a new context with the roundID
	ctxWithRoundID := context.WithValue(ctx, roundIDCtxKey("round_id"), int64(roundID))

	// Create a new message for the handler
	handlerMsg := message.NewMessage(watermill.NewUUID(), msg.Payload)

	switch handlerName {
	case "ReminderOneHourHandler", "ReminderThirtyMinutesHandler":
		// Call ReminderHandler.Handle
		reminderHandler := NewReminderHandler(h.RoundDB, h.PubSuber)
		if err := reminderHandler.Handle(ctxWithRoundID, handlerMsg); err != nil {
			return fmt.Errorf("failed to handle reminder: %w", err)
		}

	case "StartRoundEventHandler":
		// Call StartRoundHandler.Handle
		startRoundHandler := NewStartRoundHandler(h.RoundDB, h.PubSuber)
		if err := startRoundHandler.Handle(ctxWithRoundID, handlerMsg); err != nil {
			return fmt.Errorf("failed to handle start round: %w", err)
		}

	default:
		return fmt.Errorf("unknown handler: %s", handlerName)
	}

	return nil
}
