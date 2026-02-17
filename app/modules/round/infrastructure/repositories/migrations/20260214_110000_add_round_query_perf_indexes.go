package roundmigrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Adding round query performance indexes...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `
				CREATE INDEX IF NOT EXISTS idx_rounds_guild_state_start_time_desc ON rounds(guild_id, state, start_time DESC);
				CREATE INDEX IF NOT EXISTS idx_rounds_participants_gin ON rounds USING GIN (participants);
			`); err != nil {
				return fmt.Errorf("failed to create round query performance indexes: %w", err)
			}
			return nil
		})
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Rolling back round query performance indexes...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `
				DROP INDEX IF EXISTS idx_rounds_participants_gin;
				DROP INDEX IF EXISTS idx_rounds_guild_state_start_time_desc;
			`); err != nil {
				return fmt.Errorf("failed to drop round query performance indexes: %w", err)
			}
			return nil
		})
	})
}
