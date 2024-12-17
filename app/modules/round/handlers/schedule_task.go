package roundhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// TaskData represents the structure of the task payload.
type TaskData struct {
	RoundID     int64  `json:"round_id"`
	HandlerName string `json:"handler"`
	ScheduledAt string `json:"scheduled_at,omitempty"`
}

// Validate validates the TaskData.
func (t TaskData) Validate() error {
	if t.RoundID == 0 {
		return fmt.Errorf("round_id is missing or invalid")
	}
	if t.HandlerName == "" {
		return fmt.Errorf("handler is missing or invalid")
	}
	return nil
}

// ScheduledTaskHandlerError is a custom error type for scheduled task handler issues.
type ScheduledTaskHandlerError struct {
	msg string
}

func (e ScheduledTaskHandlerError) Error() string {
	return e.msg
}

// ScheduledTaskHandler handles scheduled task events.
type ScheduledTaskHandler struct {
	RoundDB    rounddb.RoundDB
	handlerMap map[string]func(context.Context, *message.Message) error
	pubsub     watermillutil.PubSuber // Add pubsub dependency
}

// NewScheduledTaskHandler creates a new ScheduledTaskHandler with dependency injection.
func NewScheduledTaskHandler(roundDB rounddb.RoundDB, handlerMap map[string]func(context.Context, *message.Message) error, pubsub watermillutil.PubSuber) *ScheduledTaskHandler {
	return &ScheduledTaskHandler{
		RoundDB:    roundDB,
		handlerMap: handlerMap,
		pubsub:     pubsub, // Initialize pubsub
	}
}

// MetadataKey is the type for context keys.
type MetadataKey string

// Exported context keys (Capitalized)
const (
	ScheduledAtKey MetadataKey = "scheduled_at"
	RoundIDKey     MetadataKey = "round_id"
	HandlerKey     MetadataKey = "handler"
)

// GetRoundIDFromContext retrieves the round ID from the context.
func GetRoundIDFromContext(ctx context.Context) (int64, bool) {
	roundID, ok := ctx.Value(RoundIDKey).(int64)
	return roundID, ok
}

// GetScheduledAtFromContext retrieves the scheduledAt from the context
func GetScheduledAtFromContext(ctx context.Context) (time.Time, bool) {
	scheduledAt, ok := ctx.Value(ScheduledAtKey).(time.Time)
	return scheduledAt, ok
}

// GetHandlerNameFromContext retrieves the handler name from the context.
func GetHandlerNameFromContext(ctx context.Context) (string, bool) {
	handlerName, ok := ctx.Value(HandlerKey).(string)
	return handlerName, ok
}

// Handle processes scheduled task events.
func (h *ScheduledTaskHandler) Handle(msg *message.Message) error {
	ctx := msg.Context() // Get the context from the message

	var taskData TaskData
	if err := json.Unmarshal(msg.Payload, &taskData); err != nil {
		return fmt.Errorf("failed to unmarshal task data: %w", err)
	}

	if err := taskData.Validate(); err != nil {
		return fmt.Errorf("invalid task data: %w", err)
	}

	// Extract scheduledAt from msg.Metadata (assuming it's stored there)
	scheduledAtStr := msg.Metadata.Get("scheduled_at")
	scheduledAt, err := time.Parse(time.RFC3339, scheduledAtStr)
	if err != nil {
		return fmt.Errorf("failed to parse scheduled_at: %w", err)
	}

	if !scheduledAt.IsZero() && time.Now().Before(scheduledAt) {
		return nil // Not time to execute the task yet
	}

	// Extract roundID from taskData (assuming it's part of the task data)
	roundID := taskData.RoundID
	ctx = context.WithValue(ctx, RoundIDKey, roundID)

	// Extract handlerName from taskData (assuming it's part of the task data)
	handlerName := taskData.HandlerName

	handler, exists := h.handlerMap[handlerName]
	if !exists {
		return ScheduledTaskHandlerError{fmt.Sprintf("unknown handler: %s", handlerName)}
	}

	return handler(ctx, msg)
}
