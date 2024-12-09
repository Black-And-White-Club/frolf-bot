package common

type RoundState string

// ScoreSubmissionEvent defines the interface for score submission events.
type ScoreSubmissionEvent interface {
	GetRoundID() int64
	GetDiscordID() string
	GetScore() int
}

// Enum constants for RoundState
const (
	RoundStateUpcoming   RoundState = "UPCOMING"
	RoundStateInProgress RoundState = "IN_PROGRESS"
	RoundStateFinalized  RoundState = "FINALIZED"
)
