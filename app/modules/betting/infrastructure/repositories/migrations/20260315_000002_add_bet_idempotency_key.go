package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Adding idempotency_key to betting_bets...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `
				ALTER TABLE betting_bets ADD COLUMN idempotency_key VARCHAR(128);
			`); err != nil {
				return fmt.Errorf("add idempotency_key column: %w", err)
			}

			if _, err := tx.ExecContext(ctx, `
				CREATE UNIQUE INDEX idx_bet_idempotency ON betting_bets (user_uuid, idempotency_key)
				    WHERE idempotency_key IS NOT NULL;
			`); err != nil {
				return fmt.Errorf("create idempotency index: %w", err)
			}

			return nil
		})
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Dropping idempotency_key from betting_bets...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `DROP INDEX IF EXISTS idx_bet_idempotency;`); err != nil {
				return fmt.Errorf("drop idempotency index: %w", err)
			}

			if _, err := tx.ExecContext(ctx, `ALTER TABLE betting_bets DROP COLUMN IF EXISTS idempotency_key;`); err != nil {
				return fmt.Errorf("drop idempotency_key column: %w", err)
			}

			return nil
		})
	})
}
