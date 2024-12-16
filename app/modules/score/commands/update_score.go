package scorecommands

import (
	scoredb "github.com/Black-And-White-Club/tcr-bot/app/modules/score/db"
	"github.com/Black-And-White-Club/tcr-bot/internal/commands"
)

// UpdateScoresCommand represents a command to update scores for a round.
type UpdateScoresCommand struct {
	RoundID string          `json:"round_id"`
	Scores  []scoredb.Score `json:"scores"`
}

// CommandName returns the command name for UpdateScoresCommand
func (cmd UpdateScoresCommand) CommandName() string {
	return "create_user"
}

var _ commands.Command = UpdateScoresCommand{}
