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

func (db *ScoreDBImpl) LogScores(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, scores []sharedtypes.ScoreInfo, source string) error {
	scoreToLog := Score{
		GuildID:   guildID,
		RoundID:   roundID,
		RoundData: scores,
		Source:    source,
	}

	res, err := db.DB.NewInsert().
		Model(&scoreToLog).
		On("CONFLICT (id) DO UPDATE").
		Set("round_data = EXCLUDED.round_data, source = EXCLUDED.source, guild_id = EXCLUDED.guild_id").
		Exec(ctx)

	fmt.Printf("[DEBUG LogScores] Insert/Update Executed for RoundID %s. Error: %v, Result: %+v\n", roundID, err, res)

	if err != nil {
		return fmt.Errorf("failed to insert/update scores for round %s: %w", roundID, err)
	}

	return nil
}

func (db *ScoreDBImpl) UpdateScore(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, userID sharedtypes.DiscordID, newScore sharedtypes.Score) error {
	var score Score
	err := db.DB.NewSelect().
		Model(&score).
		Where("id = ? AND guild_id = ?", roundID, guildID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("score record for round %s (guild %s) not found for update", roundID, guildID)
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
		Where("id = ? AND guild_id = ?", roundID, guildID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update score record for round %s (guild %s): %w", roundID, guildID, err)
	}
	return nil
}

func (db *ScoreDBImpl) UpdateOrAddScore(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, scoreInfo sharedtypes.ScoreInfo) error {
	var score Score
	err := db.DB.NewSelect().
		Model(&score).
		Where("id = ? AND guild_id = ?", roundID, guildID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Attempt self-heal: locate any row by round ID regardless of guild and set guild_id if missing
			var anyRound Score
			errByID := db.DB.NewSelect().
				Model(&anyRound).
				Where("id = ?", roundID).
				Scan(ctx)
			if errByID == nil {
				if string(anyRound.GuildID) == "" {
					// Update the existing row to attach the correct guild_id
					anyRound.GuildID = guildID
					if _, updErr := db.DB.NewUpdate().
						Model(&anyRound).
						Where("id = ?", roundID).
						Column("guild_id").
						Exec(ctx); updErr == nil {
						// Retry original lookup now that guild_id is set
						errRetry := db.DB.NewSelect().
							Model(&score).
							Where("id = ? AND guild_id = ?", roundID, guildID).
							Scan(ctx)
						if errRetry == nil {
							goto SCORE_LOADED
						}
					}
				}
				if anyRound.GuildID != "" && anyRound.GuildID != guildID {
					return fmt.Errorf("score record for round %s belongs to different guild %s", roundID, anyRound.GuildID)
				}
				score = anyRound
				goto SCORE_LOADED
			}
			if !errors.Is(errByID, sql.ErrNoRows) {
				return fmt.Errorf("failed to fetch score record by round id: %w", errByID)
			}

			// No existing row: create a new score record for manual updates
			scoreToInsert := Score{
				GuildID:   guildID,
				RoundID:   roundID,
				RoundData: []sharedtypes.ScoreInfo{scoreInfo},
				Source:    "manual",
			}
			if _, insertErr := db.DB.NewInsert().
				Model(&scoreToInsert).
				On("CONFLICT (id) DO NOTHING").
				Exec(ctx); insertErr != nil {
				return fmt.Errorf("failed to insert score record for round %s (guild %s): %w", roundID, guildID, insertErr)
			}

			errRetry := db.DB.NewSelect().
				Model(&score).
				Where("id = ? AND guild_id = ?", roundID, guildID).
				Scan(ctx)
			if errRetry != nil {
				if errors.Is(errRetry, sql.ErrNoRows) {
					return fmt.Errorf("score record not found for round %s (guild %s)", roundID, guildID)
				}
				return fmt.Errorf("failed to fetch score record after insert: %w", errRetry)
			}
			goto SCORE_LOADED
		}
		return fmt.Errorf("failed to fetch score record for add/update: %w", err)
	}

SCORE_LOADED:

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
		Where("id = ? AND guild_id = ?", roundID, guildID).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update score record for round %s (guild %s): %w", roundID, guildID, err)
	}
	return nil
}

func (db *ScoreDBImpl) GetScoresForRound(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) ([]sharedtypes.ScoreInfo, error) {
	var score Score
	err := db.DB.NewSelect().
		Model(&score).
		Where("id = ? AND guild_id = ?", roundID, guildID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to fetch scores for round %s (guild %s): %w", roundID, guildID, err)
	}
	return score.RoundData, nil
}
