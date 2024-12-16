package scoredb

import (
	"context"
)

// ScoreRepository defines the methods for interacting with the score database.
type ScoreDB interface {
	InsertScores(ctx context.Context, scores []Score) error
	UpdateScore(ctx context.Context, score *Score) error
	GetScoresForRound(ctx context.Context, roundID string) ([]Score, error)
	GetScore(ctx context.Context, discordID, roundID string) (*Score, error)
	InsertScore(ctx context.Context, score *Score) error
}
