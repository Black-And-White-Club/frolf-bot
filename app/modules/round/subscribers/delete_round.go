package roundcommands

import "github.com/Black-And-White-Club/tcr-bot/internal/commands"

// DeleteRound represents the command to delete a round.
type DeleteRoundRequest struct {
	RoundID int64 `json:"round_id"`
}

// CommandName returns the name of the command.
func (cmd DeleteRoundRequest) CommandName() string {
	return "delete_round"
}

var _ commands.Command = DeleteRoundRequest{}
