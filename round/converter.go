// round/converter.go

package round

import (
	rounddb "github.com/Black-And-White-Club/tcr-bot/round/db"
)

// RoundConverter defines the interface for converting between rounddb.Round and round.Round.
type RoundConverter interface {
	ConvertJoinRoundInputToParticipant(input JoinRoundInput) Participant
	ConvertModelRoundToStructRound(modelRound *rounddb.Round) *Round
	ConvertStructRoundToModelRound(structRound *Round) *rounddb.Round
	ConvertScheduleRoundInputToModel(input ScheduleRoundInput) rounddb.ScheduleRoundInput
	ConvertUpdateParticipantInputToParticipant(input UpdateParticipantResponseInput) rounddb.Participant
	ConvertJoinRoundInputToModelParticipant(input JoinRoundInput) rounddb.Participant
	ConvertRoundStateToModelRoundState(state RoundState) rounddb.RoundState
	ConvertEditRoundInputToModel(input EditRoundInput) rounddb.EditRoundInput
}

// DefaultRoundConverter is the default implementation of the RoundConverter interface.
type DefaultRoundConverter struct{}

// ConvertModelRoundToStructRound converts a rounddb.Round to a round.Round.
func (c *DefaultRoundConverter) ConvertModelRoundToStructRound(modelRound *rounddb.Round) *Round {
	if modelRound == nil {
		return nil
	}

	// Convert participants
	participants := make([]Participant, len(modelRound.Participants))
	for i, p := range modelRound.Participants {
		participants[i] = Participant{
			DiscordID: p.DiscordID,
			TagNumber: p.TagNumber,
			Response:  Response(p.Response),
		}
	}

	return &Round{ // This should now be the Round struct from round/models.go (or round/api_models.go)
		ID:           modelRound.ID,
		Title:        modelRound.Title,
		Location:     modelRound.Location,
		EventType:    modelRound.EventType,
		Date:         modelRound.Date,
		Time:         modelRound.Time,
		Finalized:    modelRound.Finalized,
		CreatorID:    modelRound.CreatorID,
		State:        RoundState(modelRound.State),
		Participants: participants,
		Scores:       modelRound.Scores,
	}
}

// ConvertStructRoundToModelRound converts a round.Round to a rounddb.Round.
func (c *DefaultRoundConverter) ConvertStructRoundToModelRound(structRound *Round) *rounddb.Round {
	if structRound == nil {
		return nil
	}

	// Convert participants
	participants := make([]rounddb.Participant, len(structRound.Participants))
	for i, p := range structRound.Participants {
		participants[i] = rounddb.Participant{
			DiscordID: p.DiscordID,
			TagNumber: p.TagNumber,
			Response:  rounddb.Response(p.Response), // Convert round.Response to rounddb.Response
		}
	}

	return &rounddb.Round{
		ID:           structRound.ID,
		Title:        structRound.Title,
		Location:     structRound.Location,
		EventType:    structRound.EventType,
		Date:         structRound.Date,
		Time:         structRound.Time,
		Finalized:    structRound.Finalized,
		CreatorID:    structRound.CreatorID,
		State:        rounddb.RoundState(structRound.State),
		Participants: participants,
		Scores:       structRound.Scores,
	}
}

// ConvertScheduleRoundInputToModel converts round.ScheduleRoundInput to rounddb.ScheduleRoundInput.
func (c *DefaultRoundConverter) ConvertScheduleRoundInputToModel(input ScheduleRoundInput) rounddb.ScheduleRoundInput {
	return rounddb.ScheduleRoundInput{
		Title:     input.Title,
		Location:  input.Location,
		EventType: input.EventType,
		Date:      input.Date,
		Time:      input.Time,
		DiscordID: input.DiscordID,
	}
}

func (c *DefaultRoundConverter) ConvertUpdateParticipantInputToParticipant(input UpdateParticipantResponseInput) rounddb.Participant {
	return rounddb.Participant{
		DiscordID: input.DiscordID,
		Response:  rounddb.Response(input.Response),
	}
}

// ConvertJoinRoundInputToParticipant converts JoinRoundInput to Participant.
func (c *DefaultRoundConverter) ConvertJoinRoundInputToParticipant(input JoinRoundInput) Participant {
	// Implement the conversion logic here
	return Participant{
		DiscordID: input.DiscordID,
		Response:  input.Response,
		// ... other fields as needed ...
	}
}

func (c *DefaultRoundConverter) ConvertJoinRoundInputToModelParticipant(input JoinRoundInput) rounddb.Participant {
	// Implement the conversion logic here
	return rounddb.Participant{
		DiscordID: input.DiscordID,
		Response:  rounddb.Response(input.Response), // Assuming Response is an enum
		// ... other fields as needed ...
	}
}

// ConvertRoundStateToModelRoundState converts round.RoundState to rounddb.RoundState.
func (c *DefaultRoundConverter) ConvertRoundStateToModelRoundState(state RoundState) rounddb.RoundState {
	return rounddb.RoundState(state)
}

func (c *DefaultRoundConverter) ConvertEditRoundInputToModel(input EditRoundInput) rounddb.EditRoundInput {
	return rounddb.EditRoundInput{
		Title:     input.Title,
		Location:  input.Location,
		EventType: input.EventType,
		Date:      input.Date,
		Time:      input.Time,
	}
}
