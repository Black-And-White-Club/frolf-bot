package scorehandlers

import scoredb "github.com/Black-And-White-Club/tcr-bot/app/modules/score/db"

// ScoresProcessedEvent represents sorted scores for leaderboard consumption.
type ScoresProcessedEvent struct {
	RoundID string          `json:"round_id"`
	Scores  []scoredb.Score `json:"scores"` // Use scoredb.Score directly
}

// ScoresReceivedEvent represents the event when scores are received from the round module.
type ScoresReceivedEvent struct {
	RoundID string          `json:"roundId"`
	Scores  []scoredb.Score `json:"scores"` // Use scoredb.Score directly
}

// RoundFinalizedEvent represents an event triggered when a round is finalized.
type RoundFinalizedEvent struct {
	RoundID string `json:"round_id"`
}
