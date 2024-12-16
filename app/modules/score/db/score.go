package scoredb

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/uptrace/bun"
)

// ScoreDB is an implementation of the ScoreDB interface using bun.
type ScoreDBImpl struct { // Changed from scoreDB to ScoreDB
	DB *bun.DB
}

// InsertScores inserts a batch of scores into the database as a JSON blob.
func (db *ScoreDBImpl) InsertScores(ctx context.Context, scores []Score) error {
	// Convert scores to JSON
	jsonData, err := json.Marshal(scores)
	if err != nil {
		return fmt.Errorf("failed to marshal scores: %w", err)
	}

	// Insert JSON blob into the database
	_, err = db.DB.NewInsert().
		Model(&map[string]interface{}{"round_id": scores[0].RoundID, "scores_json": jsonData}).
		Table("scores"). // Use the 'scores' table
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to insert scores: %w", err)
	}
	return nil
}

// UpdateScore updates a specific score in the database.
func (db *ScoreDBImpl) UpdateScore(ctx context.Context, score *Score) error {
	_, err := db.DB.NewUpdate().
		Model(score).
		Where("discord_id = ? AND round_id = ?", score.DiscordID, score.RoundID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update score: %w", err)
	}
	return nil
}

// GetScoresForRound retrieves all scores for a given round.
func (db *ScoreDBImpl) GetScoresForRound(ctx context.Context, roundID string) ([]Score, error) {
	// Fetch the JSON blob from the database
	var data map[string]interface{}
	err := db.DB.NewSelect().
		Column("scores_json"). // Assuming you have a column named 'scores_json' in the 'scores' table
		Table("scores").       // Use the 'scores' table
		Where("round_id = ?", roundID).
		Scan(ctx, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch scores for round: %w", err)
	}

	// Unmarshal the JSON blob into a slice of Score structs
	var scores []Score
	err = json.Unmarshal([]byte(data["scores_json"].(string)), &scores)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal scores: %w", err)
	}

	return scores, nil
}

// GetScore retrieves a specific score from the database.
func (db *ScoreDBImpl) GetScore(ctx context.Context, discordID, roundID string) (*Score, error) {
	scores, err := db.GetScoresForRound(ctx, roundID)
	if err != nil {
		return nil, fmt.Errorf("failed to get scores for round: %w", err)
	}

	for _, score := range scores {
		if score.DiscordID == discordID {
			return &score, nil
		}
	}

	return nil, nil // Not found
}

// InsertScore inserts a single score into the database.
func (db *ScoreDBImpl) InsertScore(ctx context.Context, score *Score) error {
	_, err := db.DB.NewInsert().
		Model(score).
		Table("scores").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to insert score: %w", err)
	}
	return nil
}
