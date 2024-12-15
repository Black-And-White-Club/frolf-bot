package roundhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
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
}

// NewScheduledTaskHandler creates a new ScheduledTaskHandler with dependency injection.
func NewScheduledTaskHandler(roundDB rounddb.RoundDB, handlerMap map[string]func(context.Context, *message.Message) error) *ScheduledTaskHandler {
	return &ScheduledTaskHandler{
		RoundDB:    roundDB,
		handlerMap: handlerMap,
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

// parseScheduledAt parses the "scheduled_at" metadata.
func parseScheduledAt(metadata message.Metadata) (time.Time, error) {
	scheduledAtStr := metadata.Get(string(ScheduledAtKey))
	if scheduledAtStr == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.RFC3339, scheduledAtStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse scheduled_at: %w", err)
	}
	return t, nil
}

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
func (h *ScheduledTaskHandler) Handle(ctx context.Context, msg *message.Message) error {
	var taskData TaskData
	if err := json.Unmarshal(msg.Payload, &taskData); err != nil {
		return fmt.Errorf("failed to unmarshal task data: %w", err)
	}

	if err := taskData.Validate(); err != nil {
		return fmt.Errorf("invalid task data: %w", err)
	}

	scheduledAt, ok := GetScheduledAtFromContext(ctx)
	if !ok {
		return ScheduledTaskHandlerError{"scheduledAt not found in context"}
	}

	if !scheduledAt.IsZero() && time.Now().Before(scheduledAt) {
		return nil // Not time to execute the task yet
	}

	roundID, ok := GetRoundIDFromContext(ctx)
	if !ok {
		return ScheduledTaskHandlerError{"roundId not found in context"}
	}
	ctx = context.WithValue(ctx, RoundIDKey, roundID)

	handlerName, ok := GetHandlerNameFromContext(ctx)
	if !ok {
		return ScheduledTaskHandlerError{"handlerName not found in context"}
	}

	handler, exists := h.handlerMap[handlerName]
	if !exists {
		return ScheduledTaskHandlerError{fmt.Sprintf("unknown handler: %s", handlerName)}
	}

	return handler(ctx, msg)
}
