package roundservice

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// ProcessRoundStart handles the start of a round, updates participant data, updates DB, and notifies Discord.
func (s *RoundService) ProcessRoundStart(msg *message.Message) error {
	_, eventPayload, err := eventutil.UnmarshalPayload[roundevents.RoundStartedPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	s.logger.Info("Processing round start", "round_id", eventPayload.RoundID)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Fetch the round from DB
	round, err := s.RoundDB.GetRound(ctx, eventPayload.RoundID)
	if err != nil {
		return fmt.Errorf("failed to get round from database: %w", err)
	}

	// Update the round state to "in progress"
	round.State = roundtypes.RoundStateInProgress

	if err := s.RoundDB.UpdateRound(ctx, round.ID, round); err != nil {
		return fmt.Errorf("failed to update round: %w", err)
	}

	// If there's no EventMessageID, log an error but continue processing
	if round.EventMessageID == "" {
		s.logger.Warn("Missing EventMessageID for round start", "round_id", round.ID)
	}

	// Convert []roundtypes.Participant to []roundevents.RoundParticipant
	participants := make([]roundevents.RoundParticipant, len(round.Participants))
	for i, p := range round.Participants {
		participants[i] = roundevents.RoundParticipant{
			UserID:    roundtypes.UserID(p.UserID),
			TagNumber: p.TagNumber,
			Response:  roundtypes.Response(p.Response),
			Score:     nil,
		}
	}

	// Include in event payload
	discordPayload := roundevents.DiscordRoundStartPayload{
		RoundID:        round.ID,
		Title:          round.Title,
		Location:       round.Location,
		StartTime:      round.StartTime,
		Participants:   participants,
		EventMessageID: round.EventMessageID,
	}

	// Marshal the payload
	payloadBytes, err := json.Marshal(discordPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal DiscordRoundStartPayload: %w", err)
	}

	// Publish events in the exact order expected by tests:

	// 1. First publish to discord.round.started
	discordMsg := message.NewMessage(watermill.NewUUID(), payloadBytes)
	if err := s.EventBus.Publish(roundevents.DiscordRoundStarted, discordMsg); err != nil {
		return fmt.Errorf("failed to publish round.started event: %w", err)
	}

	// 2. Then publish to discord.round.event
	if err := s.EventBus.Publish(roundevents.DiscordEventsSubject, discordMsg); err != nil {
		return fmt.Errorf("failed to publish to discord.round.event: %w", err)
	}

	// 3. Finally publish to round.state.updated
	stateUpdateMsg := message.NewMessage(watermill.NewUUID(), payloadBytes)
	if err := s.EventBus.Publish(roundevents.RoundStateUpdated, stateUpdateMsg); err != nil {
		return fmt.Errorf("failed to publish round.state.updated event: %w", err)
	}

	s.logger.Info("Round start processed and published to Discord", "round_id", eventPayload.RoundID)
	return nil

}
