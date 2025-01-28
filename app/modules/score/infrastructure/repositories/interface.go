package scoredb

import (
	"context"
)

// ScoreDB is an interface for interacting with the score database.
type ScoreDB interface {
	LogScores(ctx context.Context, roundID string, scores []Score, source string) error
	UpdateScore(ctx context.Context, roundID, discordID string, newScore int) error
	UpdateOrAddScore(ctx context.Context, score *Score) error
	GetScoresForRound(ctx context.Context, roundID string) ([]Score, error)
}
