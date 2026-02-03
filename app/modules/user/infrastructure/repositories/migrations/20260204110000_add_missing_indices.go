package usermigrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Adding missing indices for optimization...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			// user module tables
			if _, err := tx.ExecContext(ctx, `
				CREATE INDEX IF NOT EXISTS idx_magic_links_guild_id ON magic_links(guild_id);
			`); err != nil {
				return fmt.Errorf("failed to add index to magic_links: %w", err)
			}

			return nil
		})
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Rolling back missing indices...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `
				DROP INDEX IF EXISTS idx_magic_links_guild_id;
			`); err != nil {
				return fmt.Errorf("failed to drop indices: %w", err)
			}
			return nil
		})
	})
}
