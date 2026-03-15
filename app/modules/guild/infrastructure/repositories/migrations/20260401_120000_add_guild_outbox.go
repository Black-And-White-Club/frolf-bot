package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Creating guild_outbox table...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `
				CREATE TABLE IF NOT EXISTS guild_outbox (
					id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
					topic       TEXT        NOT NULL,
					payload     JSONB       NOT NULL,
					published_at TIMESTAMPTZ,
					created_at  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
				);
			`); err != nil {
				return fmt.Errorf("create guild_outbox: %w", err)
			}

			// Partial index: only unpublished rows are scanned by the forwarder.
			if _, err := tx.ExecContext(ctx, `
				CREATE INDEX IF NOT EXISTS idx_guild_outbox_unpublished
				ON guild_outbox (created_at)
				WHERE published_at IS NULL;
			`); err != nil {
				return fmt.Errorf("create idx_guild_outbox_unpublished: %w", err)
			}

			return nil
		})
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Dropping guild_outbox table...")
		_, err := db.ExecContext(ctx, "DROP TABLE IF EXISTS guild_outbox;")
		return err
	})
}
