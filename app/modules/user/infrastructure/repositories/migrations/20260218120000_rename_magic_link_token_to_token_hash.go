package usermigrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Renaming magic_links.token column to token_hash...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `
				ALTER TABLE magic_links RENAME COLUMN token TO token_hash;
			`); err != nil {
				return fmt.Errorf("failed to rename token column: %w", err)
			}
			return nil
		})
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Rolling back magic_links.token_hash column rename...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `
				ALTER TABLE magic_links RENAME COLUMN token_hash TO token;
			`); err != nil {
				return fmt.Errorf("failed to rename token_hash column: %w", err)
			}
			return nil
		})
	})
}
