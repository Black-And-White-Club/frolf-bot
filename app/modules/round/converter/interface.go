// In round/converter/interface.go

package roundconverter

import (
	"github.com/Black-And-White-Club/tcr-bot/round/common"
	rounddb "github.com/Black-And-White-Club/tcr-bot/round/db"
	apimodels "github.com/Black-And-White-Club/tcr-bot/round/models"
)

// RoundConverter defines the interface for converting between different round representations.
type RoundConverter interface {
	ConvertModelRoundToStructRound(modelRound *rounddb.Round) *apimodels.Round
	ConvertStructRoundToModelRound(structRound *apimodels.Round) *rounddb.Round
	ConvertScheduleRoundInputToModel(input apimodels.ScheduleRoundInput) rounddb.ScheduleRoundInput
	ConvertUpdateParticipantInputToParticipant(input apimodels.UpdateParticipantResponseInput) rounddb.Participant
	ConvertJoinRoundInputToParticipant(input apimodels.JoinRoundInput) apimodels.Participant
	ConvertJoinRoundInputToModelParticipant(input apimodels.JoinRoundInput) rounddb.Participant
	ConvertRoundStateToModelRoundState(state apimodels.RoundState) rounddb.RoundState
	ConvertEditRoundInputToModel(input apimodels.EditRoundInput) rounddb.EditRoundInput
	ConvertCommonRoundStateToAPI(state common.RoundState) apimodels.RoundState // Add this method
	ConvertCommonRoundStateToDB(state common.RoundState) rounddb.RoundState    // Add this method
	// ... add other conversion methods as needed
}
