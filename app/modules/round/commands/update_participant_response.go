// In app/modules/round/commands/update_participant.go

package roundcommands

import (
	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
	"github.com/Black-And-White-Club/tcr-bot/internal/commands"
)

// UpdateParticipantRequest represents a command to update a participant.
type UpdateParticipantRequest struct {
	RoundID   int64            `json:"round_id"`
	DiscordID string           `json:"discord_id"`
	Response  rounddb.Response `json:"response"`
	TagNumber *int             `json:"tag_number"`
}

// CommandName returns the name of the command.
func (cmd UpdateParticipantRequest) CommandName() string {
	return "update_participant"
}

var _ commands.Command = UpdateParticipantRequest{}
