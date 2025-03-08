package roundservice

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
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

	// Convert rounddb.Round to roundtypes.Round
	rtRound := convertDbRoundToRtRound(dbRound)

	rtRound = s.transformParticipants(rtRound)
	rtRound.State = roundtypes.RoundStateInProgress

	// Convert roundtypes.Round back to rounddb.Round
	dbRound = convertRtRoundToDbRound(rtRound)

	dbRound.State = rounddb.RoundState(roundtypes.RoundStateInProgress) // Convert RoundState

	if err := s.RoundDB.UpdateRound(ctx, dbRound.ID, dbRound); err != nil {
		return fmt.Errorf("failed to update round: %w", err)
	}

	if err := s.EventBus.Publish(roundevents.RoundStarted, msg); err != nil {
		return fmt.Errorf("failed to publish round.started event: %w", err)
	}

	discordPayload, err := s.createDiscordPayload(eventPayload, rtRound)
	if err != nil {
		return fmt.Errorf("failed to create Discord payload: %w", err)
	}

	discordMsg := message.NewMessage(watermill.NewUUID(), discordPayload)

	// Convert int64 RoundID to string
	roundIDStr := strconv.FormatInt(eventPayload.RoundID, 10)
	discordMsg.Metadata.Set("correlationID", roundIDStr) // Use the converted string

	if err := s.EventBus.Publish(roundevents.DiscordEventsSubject, discordMsg); err != nil {
		return fmt.Errorf("failed to publish to discord.round.event: %w", err)
	}

	stateUpdatedPayload := roundevents.RoundStateUpdatedPayload{
		RoundID: eventPayload.RoundID,
		State:   roundtypes.RoundState(dbRound.State), // Convert RoundState
	}

	stateUpdatedPayloadBytes, err := json.Marshal(stateUpdatedPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal RoundStateUpdatedPayload: %w", err)
	}

	stateUpdatedMsg := message.NewMessage(watermill.NewUUID(), stateUpdatedPayloadBytes)

	// Convert int64 RoundID to string
	stateUpdatedRoundIDStr := strconv.FormatInt(eventPayload.RoundID, 10)
	stateUpdatedMsg.Metadata.Set("correlationID", stateUpdatedRoundIDStr) // Use the converted string

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
		StartTime:    eventPayload.StartTime,
		Participants: discordParticipants,
	}

	discordPayloadBytes, err := json.Marshal(discordPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal DiscordRoundStartPayload: %w", err)
	}
	return discordPayloadBytes, nil
}

func convertDbRoundToRtRound(dbRound *rounddb.Round) *roundtypes.Round {
	return &roundtypes.Round{
		ID:           dbRound.ID,
		Title:        dbRound.Title,
		Description:  &dbRound.Description,
		Location:     &dbRound.Location,
		EventType:    dbRound.EventType,
		StartTime:    &dbRound.StartTime,
		Finalized:    dbRound.Finalized,
		CreatedBy:    dbRound.CreatorID,
		State:        roundtypes.RoundState(dbRound.State),
		Participants: convertDbParticipantsToRtParticipants(dbRound.Participants),
	}
}

func convertDbParticipantsToRtParticipants(dbParticipants []rounddb.Participant) []roundtypes.RoundParticipant {
	rtParticipants := make([]roundtypes.RoundParticipant, len(dbParticipants))
	for i, dbP := range dbParticipants {
		rtParticipants[i] = roundtypes.RoundParticipant{
			DiscordID: dbP.DiscordID,
			Response:  roundtypes.Response(dbP.Response),
			TagNumber: *dbP.TagNumber,
			Score:     dbP.Score,
		}
	}
	return rtParticipants
}

func convertRtRoundToDbRound(rtRound *roundtypes.Round) *rounddb.Round {
	return &rounddb.Round{
		ID:           rtRound.ID,
		Title:        rtRound.Title,
		Description:  *rtRound.Description,
		Location:     *rtRound.Location,
		EventType:    rtRound.EventType,
		StartTime:    *rtRound.StartTime,
		Finalized:    rtRound.Finalized,
		CreatorID:    rtRound.CreatedBy,
		State:        rounddb.RoundState(rtRound.State),
		Participants: convertRtParticipantsToDbParticipants(rtRound.Participants),
	}
}

func convertRtParticipantsToDbParticipants(rtParticipants []roundtypes.RoundParticipant) []rounddb.Participant {
	dbParticipants := make([]rounddb.Participant, len(rtParticipants))
	for i, rtP := range rtParticipants {
		dbParticipants[i] = rounddb.Participant{
			DiscordID: rtP.DiscordID,
			Response:  rounddb.Response(rtP.Response),
			TagNumber: &rtP.TagNumber,
			Score:     rtP.Score,
		}
	}
	return dbParticipants
}
