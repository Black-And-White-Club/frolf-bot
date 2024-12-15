// In app/modules/round/commands/get_tag.go

package roundcommands

import (
	"github.com/Black-And-White-Club/tcr-bot/internal/commands"
)

// GetTagNumberRequest represents the request to get a user's tag number.
type GetTagNumberRequest struct {
	DiscordID string `json:"discord_id"`
}

// CommandName returns the name of the command.
func (cmd GetTagNumberRequest) CommandName() string {
	return "get_tag_number"
}

var _ commands.Command = GetTagNumberRequest{}
