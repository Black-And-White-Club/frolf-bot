package roundservice

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot/app/shared/logging"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// -- Service Functions for Scheduling Round Events --

// ScheduleRoundEvents schedules the reminder and start events for the round.
func (s *RoundService) ScheduleRoundEvents(ctx context.Context, msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundStoredPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal RoundStoredPayload: %w", err)
	}

	oneHourBefore := eventPayload.Round.StartTime.Add(-1 * time.Hour)
	thirtyMinsBefore := eventPayload.Round.StartTime.Add(-30 * time.Minute)

	// Schedule one-hour reminder
	if err := s.scheduleEvent(ctx, msg, roundevents.RoundReminder, roundevents.RoundReminderPayload{
		RoundID:      eventPayload.Round.ID,
		ReminderType: "one_hour",
	}, oneHourBefore); err != nil {
		return fmt.Errorf("failed to schedule one-hour reminder: %w", err)
	}

	// Schedule 30-minute reminder
	if err := s.scheduleEvent(ctx, msg, roundevents.RoundReminder, roundevents.RoundReminderPayload{
		RoundID:      eventPayload.Round.ID,
		ReminderType: "thirty_minutes",
	}, thirtyMinsBefore); err != nil {
		return fmt.Errorf("failed to schedule thirty-minutes reminder: %w", err)
	}

	// Schedule round start
	if err := s.scheduleEvent(ctx, msg, roundevents.RoundStarted, roundevents.RoundStartedPayload{
		RoundID: eventPayload.Round.ID,
	}, eventPayload.Round.StartTime); err != nil {
		return fmt.Errorf("failed to schedule round start: %w", err)
	}

	// Publish "round.scheduled" event
	if err := s.publishEvent(msg, roundevents.RoundScheduled, roundevents.RoundScheduledPayload{
		RoundID: eventPayload.Round.ID,
	}); err != nil {
		logging.LogErrorWithMetadata(ctx, s.logger, msg, "Failed to publish round.scheduled event", map[string]interface{}{})
		return fmt.Errorf("failed to publish round.scheduled event: %w", err)
	}

	logging.LogInfoWithMetadata(ctx, s.logger, msg, "Round events scheduled", map[string]interface{}{"round_id": eventPayload.Round.ID})

	return nil
}

// -- Helper functions --
// scheduleEvent is a helper function to schedule events (used by ScheduleRoundEvents).
func (s *RoundService) scheduleEvent(ctx context.Context, msg *message.Message, eventName string, payload interface{}, scheduledTime time.Time) error {
	// We will use the correlation ID from the incoming message to propagate it to the new message
	correlationID := middleware.MessageCorrelationID(msg)

	// Calculate the delay until the scheduled time
	delay := time.Until(scheduledTime)
	if delay < 0 {
		return fmt.Errorf("scheduled time is in the past")
	}

	// Marshal the payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal event payload: %w", err)
	}

	// Create a new message with a unique UUID
	newMessage := message.NewMessage(watermill.NewUUID(), payloadBytes)

	// Set the Nats-Msg-Id for JetStream deduplication
	newMessage.Metadata.Set("Nats-Msg-Id", newMessage.UUID+"-"+eventName)

	// Propagate the correlation ID from the original message to the new message
	middleware.SetCorrelationID(correlationID, newMessage)

	// Use a goroutine to publish the message after the delay
	go func(ctx context.Context) {
		select {
		case <-time.After(delay):
			if err := s.EventBus.Publish(eventName, newMessage); err != nil {
				logging.LogErrorWithMetadata(ctx, s.logger, msg, "Failed to publish scheduled event", map[string]interface{}{
					"event": eventName,
				})
			} else {
				logging.LogInfoWithMetadata(ctx, s.logger, msg, "Published scheduled event", map[string]interface{}{
					"event": eventName,
				})
			}
		case <-ctx.Done():
			logging.LogInfoWithMetadata(ctx, s.logger, msg, "Context cancelled, not publishing scheduled event", map[string]interface{}{
				"event": eventName,
			})
		}
	}(ctx)

	return nil
}
