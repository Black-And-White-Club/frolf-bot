// round/converter.go

package roundconverter

import (
	"github.com/Black-And-White-Club/tcr-bot/round/common"
	rounddb "github.com/Black-And-White-Club/tcr-bot/round/db"
	apimodels "github.com/Black-And-White-Club/tcr-bot/round/models"
)

// DefaultRoundConverter is the default implementation of the RoundConverter interface.
type DefaultRoundConverter struct{}

// ConvertModelRoundToStructRound converts a roundrounddb.Round to a round.Round.
func (c *DefaultRoundConverter) ConvertModelRoundToStructRound(modelRound *rounddb.Round) *apimodels.Round {
	if modelRound == nil {
		return nil
	}

	// Convert participants
	participants := make([]apimodels.Participant, len(modelRound.Participants))
	for i, p := range modelRound.Participants {
		participants[i] = apimodels.Participant{
			DiscordID: p.DiscordID,
			TagNumber: p.TagNumber,
			Response:  apimodels.Response(p.Response),
		}
	}

	return &apimodels.Round{
		ID:           modelRound.ID,
		Title:        modelRound.Title,
		Location:     modelRound.Location,
		EventType:    modelRound.EventType,
		Date:         modelRound.Date,
		Time:         modelRound.Time,
		Finalized:    modelRound.Finalized,
		CreatorID:    modelRound.CreatorID,
		State:        apimodels.RoundState(modelRound.State),
		Participants: participants,
		Scores:       modelRound.Scores,
	}
}

// ConvertStructRoundToModelRound converts a round.Round to a roundrounddb.Round.
func (c *DefaultRoundConverter) ConvertStructRoundToModelRound(structRound *apimodels.Round) *rounddb.Round {
	if structRound == nil {
		return nil
	}

	// Convert participants
	participants := make([]rounddb.Participant, len(structRound.Participants))
	for i, p := range structRound.Participants {
		participants[i] = rounddb.Participant{
			DiscordID: p.DiscordID,
			TagNumber: p.TagNumber,
			Response:  rounddb.Response(p.Response),
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

// ConvertScheduleRoundInputToModel converts round.ScheduleRoundInput to roundrounddb.ScheduleRoundInput.
func (c *DefaultRoundConverter) ConvertScheduleRoundInputToModel(input apimodels.ScheduleRoundInput) rounddb.ScheduleRoundInput {
	return rounddb.ScheduleRoundInput{
		Title:     input.Title,
		Location:  input.Location,
		EventType: input.EventType,
		Date:      input.Date,
		Time:      input.Time,
		DiscordID: input.DiscordID,
	}
}

func (c *DefaultRoundConverter) ConvertUpdateParticipantInputToParticipant(input apimodels.UpdateParticipantResponseInput) rounddb.Participant {
	return rounddb.Participant{
		DiscordID: input.DiscordID,
		Response:  rounddb.Response(input.Response),
	}
}

// ConvertJoinRoundInputToParticipant converts JoinRoundInput to Participant.
func (c *DefaultRoundConverter) ConvertJoinRoundInputToParticipant(input apimodels.JoinRoundInput) apimodels.Participant {
	return apimodels.Participant{
		DiscordID: input.DiscordID,
		Response:  input.Response,
	}
}

func (c *DefaultRoundConverter) ConvertJoinRoundInputToModelParticipant(input apimodels.JoinRoundInput) rounddb.Participant {
	return rounddb.Participant{
		DiscordID: input.DiscordID,
		Response:  rounddb.Response(input.Response),
	}
}

// ConvertRoundStateToModelRoundState converts round.RoundState to roundrounddb.RoundState.
func (c *DefaultRoundConverter) ConvertRoundStateToModelRoundState(state apimodels.RoundState) rounddb.RoundState {
	return rounddb.RoundState(state)
}

func (c *DefaultRoundConverter) ConvertEditRoundInputToModel(input apimodels.EditRoundInput) rounddb.EditRoundInput {
	return rounddb.EditRoundInput{
		Title:     input.Title,
		Location:  input.Location,
		EventType: input.EventType,
		Date:      input.Date,
		Time:      input.Time,
	}
}

// ConvertCommonRoundStateToAPI converts common.RoundState to apimodels.RoundState.
func (c *DefaultRoundConverter) ConvertCommonRoundStateToAPI(state common.RoundState) apimodels.RoundState {
	switch state {
	case common.RoundStateUpcoming:
		return apimodels.RoundStateUpcoming
	case common.RoundStateInProgress:
		return apimodels.RoundStateInProgress
	case common.RoundStateFinalized:
		return apimodels.RoundStateFinalized
	default:
		return apimodels.RoundStateUpcoming
	}
}

// ConvertCommonRoundStateToDB converts common.RoundState to rounddb.RoundState.
func (c *DefaultRoundConverter) ConvertCommonRoundStateToDB(state common.RoundState) rounddb.RoundState {
	switch state {
	case common.RoundStateUpcoming:
		return rounddb.RoundStateUpcoming
	case common.RoundStateInProgress:
		return rounddb.RoundStateInProgress
	case common.RoundStateFinalized:
		return rounddb.RoundStateFinalized
	default:
		// Handle the default case appropriately (e.g., return an error or a default value)
		return rounddb.RoundStateUpcoming // Or handle the error as needed
	}
}
