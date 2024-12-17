// In app/modules/round/commands/round_state.go

package roundcommands

import (
	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
	"github.com/Black-And-White-Club/tcr-bot/internal/commands"
)

// UpdateRoundStateRequest represents a command to edit a round.
type UpdateRoundStateRequest struct {
	RoundID int64              `json:"round_id"`
	State   rounddb.RoundState `json:"state"` // Add the State field

}

// CommandName returns the name of the command.
func (cmd UpdateRoundStateRequest) CommandName() string {
	return "update_round_state"
}

var _ commands.Command = UpdateRoundStateRequest{}
