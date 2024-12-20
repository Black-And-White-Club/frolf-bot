package scoredb

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/uptrace/bun"
)

// ScoreDBImpl is an implementation of the ScoreDB interface using bun.
type ScoreDBImpl struct {
	DB *bun.DB
}

// LogScores logs scores to the database with a given source (e.g., "auto" or "manual").
func (db *ScoreDBImpl) LogScores(ctx context.Context, roundID string, scores []Score, source string) error {
	// Convert scores to JSON
	jsonData, err := json.Marshal(scores)
	if err != nil {
		return fmt.Errorf("failed to marshal scores: %w", err)
	}

	// Insert JSON blob into the database with roundID and source
	_, err = db.DB.NewInsert().
		Model(&map[string]interface{}{"round_id": roundID, "scores_json": jsonData, "source": source}).
		Table("scores").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to insert scores: %w", err)
	}
	return nil
}

// UpdateScore updates a specific score in the database.
func (db *ScoreDBImpl) UpdateScore(ctx context.Context, roundID, discordID string, newScore int) error {
	// Assuming you have a column named 'score' in the 'scores' table
	_, err := db.DB.NewUpdate().
		Table("scores").
		Set("score = ?", newScore).
		Where("discord_id = ? AND round_id = ?", discordID, roundID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update score: %w", err)
	}
	return nil
}

// UpdateOrAddScore updates a score if it exists, or adds a new score entry.
func (db *ScoreDBImpl) UpdateOrAddScore(ctx context.Context, score *Score) error {
	// Check if a score already exists for the given discordID and roundID
	exists, err := db.DB.NewSelect().
		Table("scores").
		Where("discord_id = ? AND round_id = ?", score.DiscordID, score.RoundID).
		Exists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check if score exists: %w", err)
	}

	if exists {
		// Update existing score
		_, err := db.DB.NewUpdate().
			Model(score).
			Where("discord_id = ? AND round_id = ?", score.DiscordID, score.RoundID).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to update score: %w", err)
		}
		return nil
	} else {
		// Add new score entry
		_, err = db.DB.NewInsert().
			Model(score).
			Table("scores").
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to insert new score: %w", err)
		}
		return nil
	}
}

// GetScoresForRound retrieves all scores for a given round.
func (db *ScoreDBImpl) GetScoresForRound(ctx context.Context, roundID string) ([]Score, error) {
	var scores []Score
	err := db.DB.NewSelect().
		Model(&scores).
		Where("round_id = ?", roundID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch scores for round: %w", err)
	}
	return scores, nil
}
