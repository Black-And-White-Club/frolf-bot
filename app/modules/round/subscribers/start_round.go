// In app/modules/round/commands/start_round.go

package roundcommands

import "github.com/Black-And-White-Club/tcr-bot/internal/commands"

// StartRoundRequest represents the command to start a round.
type StartRoundRequest struct {
	RoundID int64 `json:"round_id"`
}

// CommandName returns the name of the command.
func (cmd StartRoundRequest) CommandName() string {
	return "start_round"
}

var _ commands.Command = StartRoundRequest{}
