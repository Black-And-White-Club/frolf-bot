package roundservice

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// ProcessRoundStart handles the start of a round, updates participant data, updates DB, and publishes necessary events.
func (s *RoundService) ProcessRoundStart(msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundStartedPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	s.logger.Info("Processing round start", "round_id", eventPayload.RoundID)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dbRound, err := s.RoundDB.GetRound(ctx, eventPayload.RoundID)
	if err != nil {
		return fmt.Errorf("failed to get round from database: %w", err)
	}

	// Update the round state directly
	dbRound.State = roundtypes.RoundStateInProgress

	if err := s.RoundDB.UpdateRound(ctx, dbRound.ID, dbRound); err != nil {
		return fmt.Errorf("failed to update round: %w", err)
	}

	if err := s.EventBus.Publish(roundevents.RoundStarted, msg); err != nil {
		return fmt.Errorf("failed to publish round.started event: %w", err)
	}

	// Use dbRound directly in createDiscordPayload
	discordPayload, err := s.createDiscordPayload(eventPayload, dbRound) // Pass dbRound directly
	if err != nil {
		return fmt.Errorf("failed to create Discord payload: %w", err)
	}

	discordMsg := message.NewMessage(watermill.NewUUID(), discordPayload)

	// Convert int64 RoundID to string
	roundIDStr := strconv.FormatInt(int64(eventPayload.RoundID), 10)
	discordMsg.Metadata.Set("correlationID", roundIDStr) // Use the converted string

	if err := s.EventBus.Publish(roundevents.DiscordEventsSubject, discordMsg); err != nil {
		return fmt.Errorf("failed to publish to discord.round.event: %w", err)
	}

	stateUpdatedPayload := roundevents.RoundStateUpdatedPayload{
		RoundID: eventPayload.RoundID,
		State:   roundtypes.RoundState(dbRound.State), // Use the state directly
	}

	stateUpdatedPayloadBytes, err := json.Marshal(stateUpdatedPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal RoundStateUpdatedPayload: %w", err)
	}

	stateUpdatedMsg := message.NewMessage(watermill.NewUUID(), stateUpdatedPayloadBytes)

	// Convert int64 RoundID to string
	stateUpdatedRoundIDStr := strconv.FormatInt(int64(eventPayload.RoundID), 10)
	stateUpdatedMsg.Metadata.Set("correlationID", stateUpdatedRoundIDStr) // Use the converted string

	if err := s.EventBus.Publish(roundevents.RoundStateUpdated, stateUpdatedMsg); err != nil {
		return fmt.Errorf("failed to publish round.state.updated event: %w", err)
	}

	s.logger.Info("Round start processed and published to Discord", "round_id", eventPayload.RoundID)
	return nil
}

// createDiscordPayload constructs the payload to send to the Discord bot.
func (s *RoundService) createDiscordPayload(eventPayload roundevents.RoundStartedPayload, round *roundtypes.Round) ([]byte, error) {
	discordParticipants := make([]roundevents.RoundParticipant, 0, len(round.Participants))
	for _, p := range round.Participants {
		discordParticipants = append(discordParticipants, roundevents.RoundParticipant{
			UserID:    p.UserID,
			TagNumber: p.TagNumber,
			Response:  p.Response,
		})
	}

	discordPayload := roundevents.DiscordRoundStartPayload{
		RoundID:      eventPayload.RoundID,
		Title:        eventPayload.Title,
		Location:     eventPayload.Location,
		StartTime:    eventPayload.StartTime,
		Participants: discordParticipants,
	}

	discordPayloadBytes, err := json.Marshal(discordPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal DiscordRoundStartPayload: %w", err)
	}
	return discordPayloadBytes, nil
}
