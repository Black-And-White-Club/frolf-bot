package roundcommands

import (
	rounddto "github.com/Black-And-White-Club/tcr-bot/app/modules/round/dto"
	"github.com/Black-And-White-Club/tcr-bot/internal/commands"
)

type JoinRoundRequest struct {
	RoundID  int64             `json:"round_id"`
	Response rounddto.Response `json:"response"`
}

func (cmd JoinRoundRequest) CommandName() string {
	return "join_round"
}

var _ commands.Command = JoinRoundRequest{}
