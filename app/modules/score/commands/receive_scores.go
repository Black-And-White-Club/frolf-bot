package scorecommands

import "github.com/Black-And-White-Club/tcr-bot/internal/commands"

// SubmitScoreRequest represents the command to submit scores for a round.
type SubmitScoreRequest struct {
	RoundID string            `json:"roundId"`
	Scores  []IndividualScore `json:"scores"`
}

// IndividualScore represents the score of an individual participant.
type IndividualScore struct {
	UserID    string `json:"userId"`
	Score     int    `json:"score"`
	TagNumber int    `json:"tagNumber"`
}

// CommandName returns the command name for SubmitScoreRequest
func (cmd SubmitScoreRequest) CommandName() string {
	return "create_user"
}

var _ commands.Command = SubmitScoreRequest{}
