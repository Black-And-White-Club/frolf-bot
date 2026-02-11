package leaderboarddb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// RoundOutcomeRepository defines operations on the leaderboard_round_outcomes table.
type RoundOutcomeRepository interface {
	// GetRoundOutcome retrieves a round outcome by guild and round ID.
	GetRoundOutcome(ctx context.Context, db bun.IDB, guildID string, roundID uuid.UUID) (*RoundOutcome, error)

	// UpsertRoundOutcome creates or updates a round outcome record.
	UpsertRoundOutcome(ctx context.Context, db bun.IDB, outcome *RoundOutcome) error
}

// RoundOutcomeRepo implements RoundOutcomeRepository.
type RoundOutcomeRepo struct{}

func NewRoundOutcomeRepo() RoundOutcomeRepository {
	return &RoundOutcomeRepo{}
}

func (r *RoundOutcomeRepo) GetRoundOutcome(ctx context.Context, db bun.IDB, guildID string, roundID uuid.UUID) (*RoundOutcome, error) {
	outcome := new(RoundOutcome)
	err := db.NewSelect().
		Model(outcome).
		Where("guild_id = ?", guildID).
		Where("round_id = ?", roundID).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("roundoutcome.GetRoundOutcome: %w", err)
	}
	return outcome, nil
}

func (r *RoundOutcomeRepo) UpsertRoundOutcome(ctx context.Context, db bun.IDB, outcome *RoundOutcome) error {
	_, err := db.NewInsert().
		Model(outcome).
		On("CONFLICT (guild_id, round_id) DO UPDATE").
		Set("season_id = EXCLUDED.season_id").
		Set("processing_hash = EXCLUDED.processing_hash").
		Set("processed_at = EXCLUDED.processed_at").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("roundoutcome.UpsertRoundOutcome: %w", err)
	}
	return nil
}
