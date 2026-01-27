package scoredb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/uptrace/bun"
)

// Impl implements the Repository interface using Bun ORM.
type Impl struct {
	db bun.IDB
}

// NewRepository creates a new score repository.
func NewRepository(db bun.IDB) Repository {
	return &Impl{db: db}
}

// resolveDB is a helper to use the provided DB (likely a transaction)
// or fall back to the default repository connection.
func (r *Impl) resolveDB(db bun.IDB) bun.IDB {
	if db == nil {
		return r.db
	}
	return db
}

func (r *Impl) LogScores(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, scores []sharedtypes.ScoreInfo, source string) error {
	db = r.resolveDB(db)

	scoreToLog := Score{
		GuildID:   guildID,
		RoundID:   roundID,
		RoundData: scores,
		Source:    source,
	}

	_, err := db.NewInsert().
		Model(&scoreToLog).
		On("CONFLICT (id) DO UPDATE").
		Set("round_data = EXCLUDED.round_data, source = EXCLUDED.source, guild_id = EXCLUDED.guild_id").
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("scoredb.LogScores: %w", err)
	}

	return nil
}

func (r *Impl) UpdateScore(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, userID sharedtypes.DiscordID, newScore sharedtypes.Score) error {
	db = r.resolveDB(db)

	var score Score
	err := db.NewSelect().
		Model(&score).
		Where("id = ? AND guild_id = ?", roundID, guildID).
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("scoredb.UpdateScore: %w", err)
	}

	for i, scoreInfo := range score.RoundData {
		if scoreInfo.UserID == userID {
			score.RoundData[i].Score = newScore
			break
		}
	}

	_, err = db.NewUpdate().
		Model(&score).
		Where("id = ? AND guild_id = ?", roundID, guildID).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("scoredb.UpdateScore: %w", err)
	}
	return nil
}

func (r *Impl) UpdateOrAddScore(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, scoreInfo sharedtypes.ScoreInfo) error {
	db = r.resolveDB(db)

	var score Score
	err := db.NewSelect().
		Model(&score).
		Where("id = ? AND guild_id = ?", roundID, guildID).
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Handle self-heal logic
			var anyRound Score
			errByID := db.NewSelect().
				Model(&anyRound).
				Where("id = ?", roundID).
				Scan(ctx)

			if errByID == nil {
				if string(anyRound.GuildID) == "" {
					anyRound.GuildID = guildID
					if _, updErr := db.NewUpdate().
						Model(&anyRound).
						Where("id = ?", roundID).
						Column("guild_id").
						Exec(ctx); updErr == nil {
						// Retry lookup
						errRetry := db.NewSelect().
							Model(&score).
							Where("id = ? AND guild_id = ?", roundID, guildID).
							Scan(ctx)
						if errRetry == nil {
							goto SCORE_LOADED
						}
					}
				}
				if anyRound.GuildID != "" && anyRound.GuildID != guildID {
					return fmt.Errorf("scoredb.UpdateOrAddScore: round %s belongs to guild %s", roundID, anyRound.GuildID)
				}
				score = anyRound
				goto SCORE_LOADED
			}

			if !errors.Is(errByID, sql.ErrNoRows) {
				return fmt.Errorf("scoredb.UpdateOrAddScore: %w", errByID)
			}

			// Insert new record
			scoreToInsert := Score{
				GuildID:   guildID,
				RoundID:   roundID,
				RoundData: []sharedtypes.ScoreInfo{scoreInfo},
				Source:    "manual",
			}
			if _, insertErr := db.NewInsert().
				Model(&scoreToInsert).
				On("CONFLICT (id) DO NOTHING").
				Exec(ctx); insertErr != nil {
				return fmt.Errorf("scoredb.UpdateOrAddScore: %w", insertErr)
			}

			// Final reload
			if errRetry := db.NewSelect().Model(&score).Where("id = ? AND guild_id = ?", roundID, guildID).Scan(ctx); errRetry != nil {
				return fmt.Errorf("scoredb.UpdateOrAddScore retry: %w", errRetry)
			}
			goto SCORE_LOADED
		}
		return fmt.Errorf("scoredb.UpdateOrAddScore: %w", err)
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

	_, err = db.NewUpdate().
		Model(&score).
		Where("id = ? AND guild_id = ?", roundID, guildID).
		Exec(ctx)

	if err != nil {
		return fmt.Errorf("scoredb.UpdateOrAddScore: %w", err)
	}
	return nil
}

func (r *Impl) GetScoresForRound(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) ([]sharedtypes.ScoreInfo, error) {
	db = r.resolveDB(db)

	var score Score
	err := db.NewSelect().
		Model(&score).
		Where("id = ? AND guild_id = ?", roundID, guildID).
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("scoredb.GetScoresForRound: %w", err)
	}
	return score.RoundData, nil
}
