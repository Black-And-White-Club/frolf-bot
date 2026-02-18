package usermigrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Adding access_token fields to linked_identities...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `
				ALTER TABLE linked_identities
				ADD COLUMN IF NOT EXISTS access_token TEXT,
				ADD COLUMN IF NOT EXISTS access_token_expires_at TIMESTAMPTZ;
			`); err != nil {
				return fmt.Errorf("failed to add access_token columns: %w", err)
			}
			return nil
		})
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Rolling back access_token fields from linked_identities...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `
				ALTER TABLE linked_identities
				DROP COLUMN IF EXISTS access_token,
				DROP COLUMN IF EXISTS access_token_expires_at;
			`); err != nil {
				return fmt.Errorf("failed to drop access_token columns: %w", err)
			}
			return nil
		})
	})
}
