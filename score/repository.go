package bundb

import (
	"context"
	"fmt"

	"github.com/Black-And-White-Club/tcr-bot/models"
	"github.com/uptrace/bun"
)

// scoreDB is an implementation of the Score interface using bun.
type scoreDB struct {
	db *bun.DB
}

// GetUserScore retrieves the score for a specific user and round.
func (db *scoreDB) GetUserScore(ctx context.Context, discordID, roundID string) (*models.Score, error) {
	var score models.Score
	err := db.db.NewSelect().
		Model(&score).
		Where("discord_id = ? AND round_id = ?", discordID, roundID).
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch user score: %w", err)
	}

	return &score, nil
}

// GetScoresForRound retrieves all scores for a given round.
func (db *scoreDB) GetScoresForRound(ctx context.Context, roundID string) ([]models.Score, error) {
	var scores []models.Score
	err := db.db.NewSelect().
		Model(&scores).
		Where("round_id = ?", roundID).
		Scan(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch scores for round: %w", err)
	}

	return scores, nil
}

// ProcessScores inserts a batch of scores into the database.
func (db *scoreDB) ProcessScores(ctx context.Context, roundID int64, scores []models.Score) error {
	_, err := db.db.NewInsert().
		Model(&scores).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to insert scores: %w", err)
	}
	return nil
}

// UpdateScore updates a specific score.
func (db *scoreDB) UpdateScore(ctx context.Context, score *models.Score) error {
	_, err := db.db.NewUpdate().
		Model(score).
		Where("discord_id = ? AND round_id = ?", score.DiscordID, score.RoundID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update score: %w", err)
	}
	return nil
}
