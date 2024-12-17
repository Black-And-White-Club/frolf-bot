// In app/modules/round/commands/record_scores.go

package roundcommands

import "github.com/Black-And-White-Club/tcr-bot/internal/commands"

// RecordScoresRequest represents the command to record scores for a round.
type RecordScoresRequest struct {
	RoundID int64 `json:"round_id"`
}

// CommandName returns the name of the command.
func (cmd RecordScoresRequest) CommandName() string {
	return "record_scores"
}

var _ commands.Command = RecordScoresRequest{}
