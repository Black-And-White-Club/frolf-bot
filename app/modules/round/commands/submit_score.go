// In app/modules/round/commands/submit_score.go

package roundcommands

import "github.com/Black-And-White-Club/tcr-bot/internal/commands"

// SubmitScoreRequest represents the command to submit a score for a participant in a round.
type SubmitScoreRequest struct {
	RoundID       int64  `json:"round_id"`
	ParticipantID string `json:"participant_id"`
	Score         int    `json:"score"`
}

// CommandName returns the name of the command.
func (cmd SubmitScoreRequest) CommandName() string {
	return "update_round_state"
}

var _ commands.Command = SubmitScoreRequest{}
