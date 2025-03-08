package roundservice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"log/slog"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot/app/shared/logging"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// ScheduleRoundEvents schedules a 1-hour reminder and the start event for the round.
func (s *RoundService) ScheduleRoundEvents(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundStoredPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal RoundScheduledPayload: %w", err)
	}

	s.logger.Info("ScheduleRoundEvents: Received RoundStored event", slog.Int64("round_id", eventPayload.Round.ID))

	// Ensure StartTime is treated as UTC
	startTime := eventPayload.Round.StartTime.UTC()

	if time.Until(startTime) < 0 {
		s.logger.Warn("Round start time is in the past, scheduling for immediate execution", "round_id", eventPayload.Round.ID)
	}

	// Prepare reusable JSON encoder & buffer for performance optimization
	var payloadBuf bytes.Buffer
	encoder := json.NewEncoder(&payloadBuf)

	// --- Schedule 1-Hour Reminder ---
	payloadBuf.Reset()
	reminderPayload := roundevents.RoundReminderPayload{
		RoundID:      eventPayload.Round.ID,
		ReminderType: "1h",
		RoundTitle:   eventPayload.Round.Title,
	}

	if err := encoder.Encode(reminderPayload); err != nil {
		s.logger.Error("Failed to encode reminder payload", "error", err)
		return fmt.Errorf("failed to encode reminder payload: %w", err)
	}

	payloadBytes := payloadBuf.Bytes()
	reminderMsg := message.NewMessage(watermill.NewUUID(), payloadBytes)
	reminderMsg.Metadata.Set("Execute-At", startTime.Add(-1*time.Hour).UTC().Format(time.RFC3339))
	reminderMsg.Metadata.Set("Original-Subject", roundevents.RoundReminder)
	reminderMsg.Metadata.Set("Nats-Msg-Id", fmt.Sprintf("%d-1h-reminder-%d", eventPayload.Round.ID, time.Now().Unix()))

	// Publish reminder message with correct subject
	reminderSubject := fmt.Sprintf("%s.%d", eventbus.DelayedMessagesSubject, eventPayload.Round.ID)
	s.logger.InfoContext(ctx, " Publishing delayed reminder message", slog.String("subject", reminderSubject), slog.Int64("round_id", eventPayload.Round.ID), slog.Time("execute_at", startTime.Add(-1*time.Hour).UTC()))
	if err := s.EventBus.Publish(reminderSubject, reminderMsg); err != nil {
		s.logger.Error("Failed to schedule reminder", "error", err, "round_id", eventPayload.Round.ID, "reminder_type", "1h")
		return fmt.Errorf("failed to schedule 1h reminder: %w", err)
	}

	// --- Schedule Round Start ---
	payloadBuf.Reset()
	startPayload := roundevents.RoundStartedPayload{
		RoundID:   eventPayload.Round.ID,
		Title:     eventPayload.Round.Title,
		Location:  eventPayload.Round.Location,
		StartTime: &startTime,
	}

	if err := encoder.Encode(startPayload); err != nil {
		s.logger.Error("Failed to encode round start payload", "error", err)
		return fmt.Errorf("failed to encode round start payload: %w", err)
	}

	startPayloadBytes := payloadBuf.Bytes()
	startMsg := message.NewMessage(watermill.NewUUID(), startPayloadBytes)
	startMsg.Metadata.Set("Execute-At", startTime.UTC().Format(time.RFC3339))
	startMsg.Metadata.Set("Original-Subject", roundevents.RoundStarted)
	startMsg.Metadata.Set("Nats-Msg-Id", fmt.Sprintf("%d-round-start-%d", eventPayload.Round.ID, time.Now().Unix()))

	// Schedule the round processing
	s.EventBus.ScheduleRoundProcessing(ctx, fmt.Sprintf("%d", eventPayload.Round.ID), startTime.UTC())
	s.logger.Info("ScheduleRoundEvents: Round processing scheduled", slog.Int64("round_id", eventPayload.Round.ID), slog.Time("execute_at", startTime.UTC()))

	// Determine if this is an initial creation or an update
	eventType := msg.Metadata.Get("event_type")
	var publishTopic string
	switch eventType {
	case roundevents.RoundCreateRequest:
		publishTopic = roundevents.RoundScheduled
	case roundevents.RoundUpdateRequest:
		publishTopic = roundevents.RoundScheduleUpdate
	default:
		s.logger.Warn("Unknown event_type metadata, defaulting to RoundScheduled", "round_id", eventPayload.Round.ID, "event_type", eventType)
		publishTopic = roundevents.RoundScheduled // Default to creation event
	}

	// Validate publish topic
	if publishTopic == "" {
		return fmt.Errorf("missing publish topic for round scheduling, round_id: %d", eventPayload.Round.ID)
	}

	// Publish final event
	scheduledMsg := roundevents.RoundScheduledPayload{
		RoundID:     eventPayload.Round.ID,
		StartTime:   eventPayload.Round.StartTime,
		Title:       eventPayload.Round.Title,
		Description: eventPayload.Round.Description,
		Location:    eventPayload.Round.Location,
		CreatedBy:   eventPayload.Round.CreatedBy,
	}

	if err := s.publishEvent(msg, publishTopic, scheduledMsg); err != nil {
		logging.LogErrorWithMetadata(ctx, s.logger, msg, fmt.Sprintf("Failed to publish %s event", publishTopic), nil)
		return fmt.Errorf("failed to publish %s event: %w", publishTopic, err)
	}

	s.logger.Info("ScheduleRoundEvents: Round events scheduled successfully", slog.Int64("round_id", eventPayload.Round.ID))
	return nil
}
