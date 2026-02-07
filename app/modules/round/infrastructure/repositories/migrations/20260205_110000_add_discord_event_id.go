package roundmigrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Adding discord_event_id column to rounds table...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			// Add the discord_event_id column
			if _, err := tx.ExecContext(ctx, `
				ALTER TABLE rounds ADD COLUMN IF NOT EXISTS discord_event_id VARCHAR DEFAULT NULL
			`); err != nil {
				return fmt.Errorf("failed to add discord_event_id column: %w", err)
			}

			// Add partial index on discord_event_id (only non-null values)
			if _, err := tx.ExecContext(ctx, `
				CREATE INDEX IF NOT EXISTS idx_rounds_discord_event_id ON rounds (discord_event_id) WHERE discord_event_id IS NOT NULL
			`); err != nil {
				return fmt.Errorf("failed to create discord_event_id index: %w", err)
			}

			fmt.Println("discord_event_id column and index added successfully!")
			return nil
		})
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Rolling back discord_event_id column from rounds table...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			// Drop the index first
			if _, err := tx.ExecContext(ctx, `
				DROP INDEX IF EXISTS idx_rounds_discord_event_id
			`); err != nil {
				return fmt.Errorf("failed to drop discord_event_id index: %w", err)
			}

			// Drop the column
			if _, err := tx.ExecContext(ctx, `
				ALTER TABLE IF EXISTS rounds DROP COLUMN IF EXISTS discord_event_id
			`); err != nil {
				return fmt.Errorf("failed to drop discord_event_id column: %w", err)
			}

			fmt.Println("discord_event_id column and index removed successfully!")
			return nil
		})
	})
}
