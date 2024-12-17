package roundhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	roundcommands "github.com/Black-And-White-Club/tcr-bot/app/modules/round/commands"
	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
	roundservice "github.com/Black-And-White-Club/tcr-bot/app/modules/round/service"
	"github.com/Black-And-White-Club/tcr-bot/internal/jetstream"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// EditRoundHandler handles the EditRound command.
type EditRoundHandler struct {
	roundDB      rounddb.RoundDB
	messageBus   watermillutil.Publisher
	roundService roundservice.Service
}

// NewEditRoundHandler creates a new EditRoundHandler.
func NewEditRoundHandler(roundDB rounddb.RoundDB, messageBus watermillutil.Publisher, roundService roundservice.Service) *EditRoundHandler {
	return &EditRoundHandler{
		roundDB:      roundDB,
		messageBus:   messageBus,
		roundService: roundService,
	}
}

// Handle processes the EditRound command.
func (h *EditRoundHandler) Handle(ctx context.Context, msg *message.Message) error {
	var cmd roundcommands.EditRoundRequest
	if err := json.Unmarshal(msg.Payload, &cmd); err != nil {
		return fmt.Errorf("failed to unmarshal EditRoundRequest: %w", err)
	}

	// Fetch the original round data before updating
	originalRound, err := h.roundDB.GetRound(ctx, cmd.RoundID)
	if err != nil {
		return fmt.Errorf("failed to get original round: %w", err)
	}

	// Construct the updates map
	updates := map[string]interface{}{
		"title":      cmd.APIInput.Title,
		"location":   cmd.APIInput.Location,
		"event_type": cmd.APIInput.EventType,
		"date":       cmd.APIInput.Date,
		"time":       cmd.APIInput.Time,
	}

	err = h.roundDB.UpdateRound(ctx, cmd.RoundID, updates)
	if err != nil {
		return fmt.Errorf("failed to update round: %w", err)
	}

	// 1. Check if the round is upcoming
	isUpcoming, err := h.roundService.IsRoundUpcoming(ctx, cmd.RoundID)
	if err != nil {
		return err
	}
	if !isUpcoming {
		return fmt.Errorf("cannot edit round that is not upcoming")
	}

	// Publish RoundEditedEvent
	event := RoundEditedEvent{
		RoundID: cmd.RoundID,
		Updates: updates,
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal RoundEditedEvent: %w", err)
	}
	if err := h.messageBus.Publish(TopicRoundEdited, message.NewMessage(watermill.NewUUID(), payload)); err != nil {
		return fmt.Errorf("failed to publish RoundEditedEvent: %w", err)
	}

	// Get the JetStream context
	js := h.messageBus.(watermillutil.PubSuber).JetStreamContext()

	// Fetch scheduled messages for the round
	fetchedMessages, err := jetstream.FetchMessagesForRound(js, cmd.RoundID)
	if err != nil {
		return fmt.Errorf("failed to fetch scheduled messages: %w", err)
	}

	// Check if start time (Date or Time) has changed
	if cmd.APIInput.Date != originalRound.Date || cmd.APIInput.Time != originalRound.Time {
		for _, msg := range fetchedMessages {
			// Extract reminderType from message
			var taskData map[string]interface{}
			if err := json.Unmarshal(msg.Data, &taskData); err != nil { // Use msg.Data here
				return fmt.Errorf("failed to unmarshal task data: %w", err)
			}
			reminderType, ok := taskData["reminder_type"].(string)
			if !ok {
				return fmt.Errorf("invalid reminder_type")
			}

			// Recalculate reminder time
			newReminderTime := calculateReminderTime(cmd.APIInput.Date, cmd.APIInput.Time, reminderType)

			// Update scheduled_at header
			msg.Header.Set("scheduled_at", newReminderTime.Format(time.RFC3339))

			// Republish the message
			_, err := js.PublishMsg(msg)
			if err != nil {
				return fmt.Errorf("failed to republish message: %w", err)
			}
		}
	}

	return nil
}
