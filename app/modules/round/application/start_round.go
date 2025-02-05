package roundservice

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot/app/modules/round/domain/types"
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

	round, err := s.RoundDB.GetRound(ctx, eventPayload.RoundID)
	if err != nil {
		return fmt.Errorf("failed to get round from database: %w", err)
	}

	round = s.transformParticipants(round)
	round.State = roundtypes.RoundStateInProgress

	if err := s.RoundDB.UpdateRound(ctx, round.ID, round); err != nil {
		return fmt.Errorf("failed to update round: %w", err)
	}

	if err := s.EventBus.Publish(roundevents.RoundStarted, msg); err != nil {
		return fmt.Errorf("failed to publish round.started event: %w", err)
	}

	discordPayload, err := s.createDiscordPayload(eventPayload, round)
	if err != nil {
		return fmt.Errorf("failed to create Discord payload: %w", err)
	}

	discordMsg := message.NewMessage(watermill.NewUUID(), discordPayload)
	discordMsg.Metadata.Set("correlationID", eventPayload.RoundID)

	if err := s.EventBus.Publish(roundevents.DiscordEventsSubject, discordMsg); err != nil {
		return fmt.Errorf("failed to publish to discord.round.event: %w", err)
	}

	stateUpdatedPayload := roundevents.RoundStateUpdatedPayload{
		RoundID: eventPayload.RoundID,
		State:   round.State,
	}

	stateUpdatedPayloadBytes, err := json.Marshal(stateUpdatedPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal RoundStateUpdatedPayload: %w", err)
	}

	stateUpdatedMsg := message.NewMessage(watermill.NewUUID(), stateUpdatedPayloadBytes)
	stateUpdatedMsg.Metadata.Set("correlationID", eventPayload.RoundID)

	if err := s.EventBus.Publish(roundevents.RoundStateUpdated, stateUpdatedMsg); err != nil {
		return fmt.Errorf("failed to publish round.state.updated event: %w", err)
	}

	s.logger.Info("Round start processed and published to Discord", "round_id", eventPayload.RoundID)
	return nil
}

// transformParticipants initializes each participant's score to 0 and retains existing details.
func (s *RoundService) transformParticipants(round *roundtypes.Round) *roundtypes.Round {
	updatedParticipants := make([]roundtypes.RoundParticipant, 0, len(round.Participants))
	for _, participant := range round.Participants {
		transformedParticipant := roundtypes.RoundParticipant{
			DiscordID: participant.DiscordID,
			Response:  participant.Response,
			TagNumber: participant.TagNumber, // Include the tag number (even if 0)
		}

		zero := 0                            // Initialize zero
		transformedParticipant.Score = &zero // Assign to Score
		updatedParticipants = append(updatedParticipants, transformedParticipant)
	}
	round.Participants = updatedParticipants
	return round
}

// createDiscordPayload constructs the payload to send to the Discord bot.
func (s *RoundService) createDiscordPayload(eventPayload roundevents.RoundStartedPayload, round *roundtypes.Round) ([]byte, error) {
	discordParticipants := make([]roundevents.DiscordRoundParticipant, 0, len(round.Participants))
	for _, p := range round.Participants {
		discordParticipants = append(discordParticipants, roundevents.DiscordRoundParticipant{
			DiscordID: p.DiscordID,
			TagNumber: p.TagNumber,
			Score:     p.Score,
		})
	}

	discordPayload := roundevents.DiscordRoundStartPayload{
		RoundID:      eventPayload.RoundID,
		Title:        eventPayload.Title,
		Location:     eventPayload.Location,
		StartTime:    eventPayload.StartTime.Format(time.RFC3339),
		Participants: discordParticipants,
	}

	discordPayloadBytes, err := json.Marshal(discordPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal DiscordRoundStartPayload: %w", err)
	}
	return discordPayloadBytes, nil
}
