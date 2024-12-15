// In app/modules/round/commands/finalize_round.go

package roundcommands

import (
	"github.com/Black-And-White-Club/tcr-bot/internal/commands"
)

// FinalizeRoundRequest represents the command to finalize a round.
type FinalizeRoundRequest struct {
	RoundID int64 `json:"round_id"`
}

// CommandName returns the name of the command.
func (cmd FinalizeRoundRequest) CommandName() string {
	return "finalize_round"
}

var _ commands.Command = FinalizeRoundRequest{}
