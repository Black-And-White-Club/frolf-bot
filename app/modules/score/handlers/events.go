package scorehandlers

import scoredto "github.com/Black-And-White-Club/tcr-bot/app/modules/score/dto"

// ScoresProcessedEvent represents sorted scores for leaderboard consumption.
type ScoresProcessedEvent struct {
	RoundID string              `json:"round_id"`
	Scores  []scoredto.ScoreDTO `json:"scores"`
}
