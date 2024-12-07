// round/converter.go

package round

import (
	"github.com/Black-And-White-Club/tcr-bot/models"
)

// RoundConverter defines the interface for converting between models.Round and round.Round.
type RoundConverter interface {
	ConvertJoinRoundInputToParticipant(input JoinRoundInput) Participant
	ConvertModelRoundToStructRound(modelRound *models.Round) *Round
	ConvertStructRoundToModelRound(structRound *Round) *models.Round
	ConvertScheduleRoundInputToModel(input ScheduleRoundInput) models.ScheduleRoundInput
	ConvertUpdateParticipantInputToParticipant(input UpdateParticipantResponseInput) models.Participant
	ConvertJoinRoundInputToModelParticipant(input JoinRoundInput) models.Participant
	ConvertRoundStateToModelRoundState(state RoundState) models.RoundState
	ConvertEditRoundInputToModel(input EditRoundInput) models.EditRoundInput
}

// DefaultRoundConverter is the default implementation of the RoundConverter interface.
type DefaultRoundConverter struct{}

// ConvertModelRoundToStructRound converts a models.Round to a round.Round.
func (c *DefaultRoundConverter) ConvertModelRoundToStructRound(modelRound *models.Round) *Round {
	if modelRound == nil {
		return nil
	}

	// Convert participants
	participants := make([]Participant, len(modelRound.Participants))
	for i, p := range modelRound.Participants {
		participants[i] = Participant{
			DiscordID: p.DiscordID,
			TagNumber: p.TagNumber,
			Response:  Response(p.Response), // Convert models.Response to round.Response
		}
	}

	return &Round{
		ID:           modelRound.ID,
		Title:        modelRound.Title,
		Location:     modelRound.Location,
		EventType:    modelRound.EventType,
		Date:         modelRound.Date,
		Time:         modelRound.Time,
		Finalized:    modelRound.Finalized,
		CreatorID:    modelRound.CreatorID,
		State:        RoundState(modelRound.State), // Convert models.RoundState to round.RoundState
		Participants: participants,
		Scores:       modelRound.Scores,
	}
}

// ConvertStructRoundToModelRound converts a round.Round to a models.Round.
func (c *DefaultRoundConverter) ConvertStructRoundToModelRound(structRound *Round) *models.Round {
	if structRound == nil {
		return nil
	}

	// Convert participants
	participants := make([]models.Participant, len(structRound.Participants))
	for i, p := range structRound.Participants {
		participants[i] = models.Participant{
			DiscordID: p.DiscordID,
			TagNumber: p.TagNumber,
			Response:  models.Response(p.Response), // Convert round.Response to models.Response
		}
	}

	return &models.Round{
		ID:           structRound.ID,
		Title:        structRound.Title,
		Location:     structRound.Location,
		EventType:    structRound.EventType,
		Date:         structRound.Date,
		Time:         structRound.Time,
		Finalized:    structRound.Finalized,
		CreatorID:    structRound.CreatorID,
		State:        models.RoundState(structRound.State),
		Participants: participants,
		Scores:       structRound.Scores,
	}
}

// ConvertScheduleRoundInputToModel converts round.ScheduleRoundInput to models.ScheduleRoundInput.
func (c *DefaultRoundConverter) ConvertScheduleRoundInputToModel(input ScheduleRoundInput) models.ScheduleRoundInput {
	return models.ScheduleRoundInput{
		Title:     input.Title,
		Location:  input.Location,
		EventType: input.EventType,
		Date:      input.Date,
		Time:      input.Time,
		DiscordID: input.DiscordID,
	}
}

func (c *DefaultRoundConverter) ConvertUpdateParticipantInputToParticipant(input UpdateParticipantResponseInput) models.Participant {
	return models.Participant{
		DiscordID: input.DiscordID,
		Response:  models.Response(input.Response),
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

func (c *DefaultRoundConverter) ConvertJoinRoundInputToModelParticipant(input JoinRoundInput) models.Participant {
	// Implement the conversion logic here
	return models.Participant{
		DiscordID: input.DiscordID,
		Response:  models.Response(input.Response), // Assuming Response is an enum
		// ... other fields as needed ...
	}
}

// ConvertRoundStateToModelRoundState converts round.RoundState to models.RoundState.
func (c *DefaultRoundConverter) ConvertRoundStateToModelRoundState(state RoundState) models.RoundState {
	return models.RoundState(state)
}

func (c *DefaultRoundConverter) ConvertEditRoundInputToModel(input EditRoundInput) models.EditRoundInput {
	return models.EditRoundInput{
		Title:     input.Title,
		Location:  input.Location,
		EventType: input.EventType,
		Date:      input.Date,
		Time:      input.Time,
	}
}
