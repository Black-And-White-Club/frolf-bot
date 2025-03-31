package scoredb

import (
	"context"
	"fmt"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// ScoreDBImpl is an implementation of the ScoreDB interface using bun.
type ScoreDBImpl struct {
	DB *bun.DB
}

// LogScores logs scores to the database with a given source (e.g., "auto" or "manual").
func (db *ScoreDBImpl) LogScores(ctx context.Context, roundID sharedtypes.RoundID, scores []sharedtypes.ScoreInfo, source string) error {
	// Create a new Score object
	score := Score{
		RoundID:   roundID,
		RoundData: scores,
		Source:    source,
	}

	// Insert Score object into the database
	_, err := db.DB.NewInsert().
		Model(&score).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to insert scores: %w", err)
	}
	return nil
}

// UpdateScore updates a specific score in the database.
func (db *ScoreDBImpl) UpdateScore(ctx context.Context, roundID sharedtypes.RoundID, userID sharedtypes.DiscordID, newScore sharedtypes.Score) error {
	// Retrieve the Score object for the given roundID
	var score Score
	err := db.DB.NewSelect().
		Model(&score).
		Where("round_id = ?", roundID).
		Scan(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch score: %w", err)
	}

	// Update the score for the given userID
	for i, scoreInfo := range score.RoundData {
		if scoreInfo.UserID == userID {
			score.RoundData[i].Score = newScore
			break
		}
	}

	// Update the Score object in the database
	_, err = db.DB.NewUpdate().
		Model(&score).
		Where("round_id = ?", roundID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update score: %w", err)
	}
	return nil
}

// UpdateOrAddScore updates a score if it exists, or adds a new score entry.
func (db *ScoreDBImpl) UpdateOrAddScore(ctx context.Context, roundID sharedtypes.RoundID, scoreInfo sharedtypes.ScoreInfo) error {
	// Retrieve the Score object for the given roundID
	var score Score
	err := db.DB.NewSelect().
		Model(&score).
		Where("round_id = ?", roundID).
		Scan(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch score: %w", err)
	}

	if score.RoundID == sharedtypes.RoundID(uuid.Nil) {
		// If the Score object does not exist, create a new one
		score = Score{
			RoundID:   roundID,
			RoundData: []sharedtypes.ScoreInfo{scoreInfo},
			Source:    "manual",
		}

		// Insert the new Score object into the database
		_, err = db.DB.NewInsert().
			Model(&score).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to insert new score: %w", err)
		}
		return nil
	}

	// Update the score for the given userID
	exists := false
	for i, existingScoreInfo := range score.RoundData {
		if existingScoreInfo.UserID == scoreInfo.UserID {
			score.RoundData[i] = scoreInfo
			exists = true
			break
		}
	}

	if !exists {
		score.RoundData = append(score.RoundData, scoreInfo)
	}

	// Update the Score object in the database
	_, err = db.DB.NewUpdate().
		Model(&score).
		Where("round_id = ?", roundID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update score: %w", err)
	}
	return nil
}

// GetScoresForRound retrieves all scores for a given round.
func (db *ScoreDBImpl) GetScoresForRound(ctx context.Context, roundID sharedtypes.RoundID) ([]sharedtypes.ScoreInfo, error) {
	var score Score
	err := db.DB.NewSelect().
		Model(&score).
		Where("round_id = ?", roundID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch scores for round: %w", err)
	}
	return score.RoundData, nil
}
