package roundmigrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Ensuring discord_event_id column exists and is TEXT...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			// 1. Add the column if it doesn't exist (defaulting to TEXT)
			if _, err := tx.ExecContext(ctx, `
				ALTER TABLE rounds ADD COLUMN IF NOT EXISTS discord_event_id TEXT DEFAULT NULL
			`); err != nil {
				return fmt.Errorf("failed to ensure discord_event_id column: %w", err)
			}

			// 2. Ensure it is TEXT type (fixes cases where it might be VARCHAR from previous migration)
			if _, err := tx.ExecContext(ctx, `
				ALTER TABLE rounds ALTER COLUMN discord_event_id TYPE TEXT
			`); err != nil {
				return fmt.Errorf("failed to set discord_event_id type to TEXT: %w", err)
			}

			// 3. Ensure the partial index exists
			if _, err := tx.ExecContext(ctx, `
				CREATE INDEX IF NOT EXISTS idx_rounds_discord_event_id ON rounds (discord_event_id) WHERE discord_event_id IS NOT NULL
			`); err != nil {
				return fmt.Errorf("failed to create discord_event_id index: %w", err)
			}

			fmt.Println("discord_event_id column ensured as TEXT and indexed!")
			return nil
		})
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Rolling back ensure_discord_event_id (reverting to VARCHAR if possible, or dropping)...")

		// Note: We can't easily know if we should revert to VARCHAR or drop it entirely
		// without complex logic. For safety, we will just revert the type to VARCHAR
		// which matches the previous migration's state.
		// If this migration added the column, this rollback leaves it as VARCHAR (which the prev migration expects).

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `
				ALTER TABLE rounds ALTER COLUMN discord_event_id TYPE VARCHAR
			`); err != nil {
				return fmt.Errorf("failed to revert discord_event_id type: %w", err)
			}
			return nil
		})
	})
}
