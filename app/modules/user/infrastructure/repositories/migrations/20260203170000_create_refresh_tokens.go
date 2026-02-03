package usermigrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Creating refresh_tokens table...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `
				CREATE TABLE IF NOT EXISTS refresh_tokens (
					hash            VARCHAR(64) PRIMARY KEY,
					user_uuid       UUID NOT NULL REFERENCES users(uuid) ON DELETE CASCADE,
					token_family    VARCHAR(64) NOT NULL,
					expires_at      TIMESTAMPTZ NOT NULL,
					created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
					last_used_at    TIMESTAMPTZ,
					revoked         BOOLEAN NOT NULL DEFAULT false,
					revoked_at      TIMESTAMPTZ,
					revoked_by      VARCHAR(50),
					ip_address      INET,
					user_agent      TEXT
				);
				CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_uuid ON refresh_tokens(user_uuid);
				CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token_family ON refresh_tokens(token_family);
				CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);
			`); err != nil {
				return fmt.Errorf("failed to create refresh_tokens table: %w", err)
			}
			return nil
		})
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Rolling back refresh_tokens table...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS refresh_tokens;`); err != nil {
				return fmt.Errorf("failed to drop refresh_tokens: %w", err)
			}
			return nil
		})
	})
}
