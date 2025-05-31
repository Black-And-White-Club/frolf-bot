package scoredb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/uptrace/bun"
)

type ScoreDBImpl struct {
	DB *bun.DB
}

func (db *ScoreDBImpl) LogScores(ctx context.Context, roundID sharedtypes.RoundID, scores []sharedtypes.ScoreInfo, source string) error {
	scoreToLog := Score{
		RoundID:   roundID,
		RoundData: scores,
		Source:    source,
	}

	res, err := db.DB.NewInsert().
		Model(&scoreToLog).
		On("CONFLICT (id) DO UPDATE").
		Set("round_data = EXCLUDED.round_data, source = EXCLUDED.source").
		Exec(ctx)

	fmt.Printf("[DEBUG LogScores] Insert/Update Executed for RoundID %s. Error: %v, Result: %+v\n", roundID, err, res)

	if err != nil {
		return fmt.Errorf("failed to insert/update scores for round %s: %w", roundID, err)
	}

	return nil
}

func (db *ScoreDBImpl) UpdateScore(ctx context.Context, roundID sharedtypes.RoundID, userID sharedtypes.DiscordID, newScore sharedtypes.Score) error {
	var score Score
	err := db.DB.NewSelect().
		Model(&score).
		Where("id = ?", roundID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("score record for round %s not found for update", roundID)
		}
		return fmt.Errorf("failed to fetch score record for update: %w", err)
	}

	for i, scoreInfo := range score.RoundData {
		if scoreInfo.UserID == userID {
			score.RoundData[i].Score = newScore
			break
		}
	}

	_, err = db.DB.NewUpdate().
		Model(&score).
		Where("id = ?", roundID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update score record for round %s: %w", roundID, err)
	}
	return nil
}

func (db *ScoreDBImpl) UpdateOrAddScore(ctx context.Context, roundID sharedtypes.RoundID, scoreInfo sharedtypes.ScoreInfo) error {
	var score Score
	err := db.DB.NewSelect().
		Model(&score).
		Where("id = ?", roundID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("score record not found for round %s", roundID)
		}
		return fmt.Errorf("failed to fetch score record for add/update: %w", err)
	}

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

	_, err = db.DB.NewUpdate().
		Model(&score).
		Where("id = ?", roundID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update score record for round %s: %w", roundID, err)
	}
	return nil
}

func (db *ScoreDBImpl) GetScoresForRound(ctx context.Context, roundID sharedtypes.RoundID) ([]sharedtypes.ScoreInfo, error) {
	var score Score
	err := db.DB.NewSelect().
		Model(&score).
		Where("id = ?", roundID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to fetch scores for round %s: %w", roundID, err)
	}
	return score.RoundData, nil
}
