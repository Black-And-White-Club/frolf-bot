// In app/modules/round/commands/join_round.go

package roundcommands

import (
	rounddto "github.com/Black-And-White-Club/tcr-bot/app/modules/round/dto"
	"github.com/Black-And-White-Club/tcr-bot/internal/commands"
)

// JoinRoundRequest represents the command for a participant to join a round.
type JoinRoundRequest struct {
	Input rounddto.JoinRoundInput `json:"input"`
}

// CommandName returns the name of the command.
func (cmd JoinRoundRequest) CommandName() string {
	return "join_round"
}

var _ commands.Command = JoinRoundRequest{}
