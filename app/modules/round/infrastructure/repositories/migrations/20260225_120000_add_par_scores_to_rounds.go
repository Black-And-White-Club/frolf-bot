package roundmigrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Adding par_scores column to rounds table...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `
				ALTER TABLE rounds ADD COLUMN IF NOT EXISTS par_scores JSONB;
			`); err != nil {
				return fmt.Errorf("failed to add par_scores column to rounds: %w", err)
			}

			return nil
		})
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Dropping par_scores column from rounds table...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `
				ALTER TABLE rounds DROP COLUMN IF EXISTS par_scores;
			`); err != nil {
				return fmt.Errorf("failed to drop par_scores column from rounds: %w", err)
			}

			return nil
		})
	})
}
