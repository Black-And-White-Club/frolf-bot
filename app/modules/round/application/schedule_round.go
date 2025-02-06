package roundservice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

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
		return fmt.Errorf("failed to unmarshal RoundStoredPayload: %w", err)
	}

	roundID := eventPayload.Round.ID

	// Prepare reusable JSON encoder & buffer for performance optimization
	var payloadBuf bytes.Buffer
	encoder := json.NewEncoder(&payloadBuf)

	payloadBuf.Reset() // Clear buffer for reuse

	reminderPayload := roundevents.RoundReminderPayload{
		RoundID:      roundID,
		ReminderType: "1h",
		RoundTitle:   eventPayload.Round.Title,
		StartTime:    eventPayload.Round.StartTime,
		Location:     eventPayload.Round.Location,
	}

	// Encode into buffer to reduce memory allocations
	if err := encoder.Encode(reminderPayload); err != nil {
		s.logger.Error("Failed to encode reminder payload", "error", err)
		return fmt.Errorf("failed to encode reminder payload: %w", err)
	}

	payloadBytes := payloadBuf.Bytes()
	reminderMsg := message.NewMessage(watermill.NewUUID(), payloadBytes)
	reminderMsg.Metadata.Set("correlationID", roundID)
	reminderMsg.Metadata.Set("Execute-At", eventPayload.Round.StartTime.Add(-1*time.Hour).Format(time.RFC3339))
	reminderMsg.Metadata.Set("Original-Subject", roundevents.RoundReminder)
	reminderMsg.Metadata.Set("Nats-Msg-Id", fmt.Sprintf("%s-1h-reminder", roundID))

	if err := s.EventBus.Publish(roundevents.DelayedMessagesSubject, reminderMsg); err != nil {
		s.logger.Error("Failed to schedule reminder", "error", err, "round_id", roundID, "reminder_type", "1h")
		return fmt.Errorf("failed to schedule 1h reminder: %w", err)
	}

	payloadBuf.Reset()

	startPayload := roundevents.RoundStartedPayload{
		RoundID:   roundID,
		Title:     eventPayload.Round.Title,
		Location:  eventPayload.Round.Location,
		StartTime: eventPayload.Round.StartTime,
	}

	if err := encoder.Encode(startPayload); err != nil {
		s.logger.Error("Failed to encode round start payload", "error", err)
		return fmt.Errorf("failed to encode round start payload: %w", err)
	}

	startPayloadBytes := payloadBuf.Bytes()
	startMsg := message.NewMessage(watermill.NewUUID(), startPayloadBytes)
	startMsg.Metadata.Set("correlationID", roundID)
	startMsg.Metadata.Set("Execute-At", eventPayload.Round.StartTime.Format(time.RFC3339))
	startMsg.Metadata.Set("Original-Subject", roundevents.RoundStarted)
	startMsg.Metadata.Set("Nats-Msg-Id", fmt.Sprintf("%s-round-start", roundID))

	if err := s.EventBus.Publish(roundevents.DelayedMessagesSubject, startMsg); err != nil {
		s.logger.Error("Failed to schedule round start", "error", err, "round_id", roundID)
		return fmt.Errorf("failed to schedule round start: %w", err)
	}

	scheduledMsg := message.NewMessage(watermill.NewUUID(), msg.Payload)
	scheduledMsg.Metadata.Set("correlationID", roundID)

	// Determine if this is an initial creation or an update
	eventType := msg.Metadata.Get("event_type")
	var publishTopic string

	if eventType == roundevents.RoundCreateRequest {
		publishTopic = roundevents.RoundScheduled
	} else if eventType == roundevents.RoundUpdateRequest {
		publishTopic = roundevents.RoundScheduleUpdate
	}

	// Publish the appropriate event
	if err := s.EventBus.Publish(publishTopic, scheduledMsg); err != nil {
		logging.LogErrorWithMetadata(ctx, s.logger, msg, fmt.Sprintf("Failed to publish %s event", publishTopic), nil)
		return fmt.Errorf("failed to publish %s event: %w", publishTopic, err)
	}

	s.logger.Info("Round events scheduled successfully", "round_id", roundID)

	return nil
}
