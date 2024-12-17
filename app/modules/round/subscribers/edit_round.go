// In app/modules/round/commands/edit_round.go

package roundcommands

import (
	rounddto "github.com/Black-And-White-Club/tcr-bot/app/modules/round/dto"
	"github.com/Black-And-White-Club/tcr-bot/internal/commands"
)

// EditRoundRequest represents a command to edit a round.
type EditRoundRequest struct {
	RoundID   int64                   `json:"round_id"`
	DiscordID string                  `json:"discord_id"`
	APIInput  rounddto.EditRoundInput `json:"api_input"`
}

// CommandName returns the name of the command.
func (cmd EditRoundRequest) CommandName() string {
	return "edit_round"
}

var _ commands.Command = EditRoundRequest{}
