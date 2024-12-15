// In app/modules/round/commands/submit_score.go

package roundcommands

import (
	rounddto "github.com/Black-And-White-Club/tcr-bot/app/modules/round/dto"
	"github.com/Black-And-White-Club/tcr-bot/internal/commands"
)

// SubmitScoreRequest represents the command to submit a score for a participant in a round.
type SubmitScoreRequest struct {
	Input rounddto.SubmitScoreInput `json:"input"`
}

// CommandName returns the name of the command.
func (cmd SubmitScoreRequest) CommandName() string {
	return "submit_score"
}

var _ commands.Command = SubmitScoreRequest{}
