// In app/modules/round/commands/process_score_submission.go

package roundcommands

import (
	rounddto "github.com/Black-And-White-Club/tcr-bot/app/modules/round/dto"
	"github.com/Black-And-White-Club/tcr-bot/internal/commands"
)

// ProcessScoreSubmissionRequest represents the command to process a score submission.
type ProcessScoreSubmissionRequest struct {
	Input rounddto.SubmitScoreInput `json:"input"`
}

// CommandName returns the name of the command.
func (cmd ProcessScoreSubmissionRequest) CommandName() string {
	return "process_score_submission"
}

var _ commands.Command = ProcessScoreSubmissionRequest{}
