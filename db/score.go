// internal/db/score.go
package db

import (
	"context"

	"github.com/Black-And-White-Club/tcr-bot/models"
)

// ScoreStore is an interface for score-related database operations.
type ScoreDB interface {
	GetUserScore(ctx context.Context, discordID, roundID string) (*models.Score, error)
	GetScoresForRound(ctx context.Context, roundID string) ([]models.Score, error)
	ProcessScores(ctx context.Context, roundID int64, scores []models.Score) error
	UpdateScore(context.Context, *models.Score) error
}
