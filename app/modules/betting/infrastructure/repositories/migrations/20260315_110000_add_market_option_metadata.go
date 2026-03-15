package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Adding metadata column to betting_market_options...")
		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `
				ALTER TABLE betting_market_options
				ADD COLUMN IF NOT EXISTS metadata TEXT NOT NULL DEFAULT '';
			`); err != nil {
				return fmt.Errorf("add metadata column: %w", err)
			}
			return nil
		})
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Dropping metadata column from betting_market_options...")
		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `ALTER TABLE betting_market_options DROP COLUMN IF EXISTS metadata;`); err != nil {
				return fmt.Errorf("drop metadata column: %w", err)
			}
			return nil
		})
	})
}
