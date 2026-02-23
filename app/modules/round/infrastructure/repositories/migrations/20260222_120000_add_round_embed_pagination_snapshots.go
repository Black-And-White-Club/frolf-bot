package roundmigrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Creating round embed pagination snapshots table...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `
				CREATE TABLE IF NOT EXISTS round_embed_pagination_snapshots (
					message_id VARCHAR PRIMARY KEY,
					snapshot_json JSONB NOT NULL,
					expires_at TIMESTAMPTZ NOT NULL,
					created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
					updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
				);
			`); err != nil {
				return fmt.Errorf("failed to create round embed pagination snapshots table: %w", err)
			}

			if _, err := tx.ExecContext(ctx, `
				CREATE INDEX IF NOT EXISTS idx_round_embed_pagination_snapshots_expires_at
					ON round_embed_pagination_snapshots (expires_at);
			`); err != nil {
				return fmt.Errorf("failed to create round embed pagination snapshots expires_at index: %w", err)
			}

			return nil
		})
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Dropping round embed pagination snapshots table...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `
				DROP INDEX IF EXISTS idx_round_embed_pagination_snapshots_expires_at;
			`); err != nil {
				return fmt.Errorf("failed to drop round embed pagination snapshots expires_at index: %w", err)
			}

			if _, err := tx.ExecContext(ctx, `
				DROP TABLE IF EXISTS round_embed_pagination_snapshots;
			`); err != nil {
				return fmt.Errorf("failed to drop round embed pagination snapshots table: %w", err)
			}

			return nil
		})
	})
}
