package roundservice

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// ProcessRoundReminder is a service function triggered when a reminder is due.
func (s *RoundService) ProcessRoundReminder(msg *message.Message) error {
	var payload roundevents.RoundReminderPayload
	err := json.Unmarshal(msg.Payload, &payload)
	if err != nil {
		return fmt.Errorf("failed to unmarshal RoundReminderPayload: %w", err)
	}

	s.logger.Info("Processing round reminder", "round_id", payload.RoundID, "reminder_type", payload.ReminderType)

	// Use a context with a timeout to prevent DB stalls
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Fetch the latest participant data
	round, err := s.RoundDB.GetRound(ctx, payload.RoundID)
	if err != nil {
		return fmt.Errorf("failed to get round from database: %w", err)
	}

	// Extract Discord IDs
	userIDs := make([]string, 0, len(round.Participants))
	for _, p := range round.Participants {
		userIDs = append(userIDs, p.DiscordID)
	}

	// Skip publishing if there are no participants
	if len(userIDs) == 0 {
		s.logger.Warn("Skipping Discord reminder: No participants in round", "round_id", payload.RoundID)
		return nil
	}

	// Create a new payload for the Discord bot
	discordPayload := roundevents.DiscordReminderPayload{
		RoundID:      payload.RoundID,
		RoundTitle:   payload.RoundTitle,
		UserIDs:      userIDs,
		ReminderType: payload.ReminderType,
	}

	discordPayloadBytes, err := json.Marshal(discordPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal DiscordReminderPayload: %w", err)
	}

	// Publish to discord.round.event with deduplication
	discordMsg := message.NewMessage(watermill.NewUUID(), discordPayloadBytes)
	discordMsg.Metadata.Set("Nats-Msg-Id", fmt.Sprintf("%d-%s-reminder", payload.RoundID, payload.ReminderType))

	if err := s.EventBus.Publish(roundevents.DiscordEventsSubject, discordMsg); err != nil {
		return fmt.Errorf("failed to publish to discord.round.event: %w", err)
	}

	s.logger.Info("Round reminder processed and published to Discord", "round_id", payload.RoundID, "reminder_type", payload.ReminderType)
	return nil
}
