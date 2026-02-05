package usermigrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Creating magic_links table...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `
				CREATE TABLE IF NOT EXISTS magic_links (
					token           VARCHAR(64) PRIMARY KEY,
					user_uuid       UUID NOT NULL REFERENCES users(uuid) ON DELETE CASCADE,
					guild_id        VARCHAR(32) NOT NULL,
					role            VARCHAR(50) NOT NULL,
					expires_at      TIMESTAMPTZ NOT NULL,
					created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
					used            BOOLEAN NOT NULL DEFAULT false,
					used_at         TIMESTAMPTZ
				);
				CREATE INDEX IF NOT EXISTS idx_magic_links_user_uuid ON magic_links(user_uuid);
				CREATE INDEX IF NOT EXISTS idx_magic_links_expires_at ON magic_links(expires_at);
			`); err != nil {
				return fmt.Errorf("failed to create magic_links table: %w", err)
			}
			return nil
		})
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Rolling back magic_links table...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS magic_links;`); err != nil {
				return fmt.Errorf("failed to drop magic_links: %w", err)
			}
			return nil
		})
	})
}
