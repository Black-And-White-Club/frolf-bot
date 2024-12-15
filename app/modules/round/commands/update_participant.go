// In app/modules/round/commands/update_participant.go

package roundcommands

import (
	rounddto "github.com/Black-And-White-Club/tcr-bot/app/modules/round/dto"
	"github.com/Black-And-White-Club/tcr-bot/internal/commands"
)

// UpdateParticipantRequest represents a command to update a participant.
type UpdateParticipantRequest struct {
	Input rounddto.UpdateParticipantResponseInput `json:"input"`
}

// CommandName returns the name of the command.
func (cmd UpdateParticipantRequest) CommandName() string {
	return "update_participant"
}

var _ commands.Command = UpdateParticipantRequest{}
