package roundcommands

import (
	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
	"github.com/Black-And-White-Club/tcr-bot/internal/commands"
)

type JoinRoundRequest struct {
	RoundID   int64            `json:"round_id"`
	DiscordID string           `json:"discord_id"`
	Response  rounddb.Response `json:"response"`
}

func (cmd JoinRoundRequest) CommandName() string {
	return "join_round"
}

var _ commands.Command = JoinRoundRequest{}
