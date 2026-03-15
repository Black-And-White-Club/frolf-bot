package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Adding CHECK constraints to betting_wallet_balances...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `
				ALTER TABLE betting_wallet_balances
				    ADD CONSTRAINT chk_balance_non_negative CHECK (balance >= 0),
				    ADD CONSTRAINT chk_available_non_negative CHECK (balance - reserved >= 0);
			`); err != nil {
				return fmt.Errorf("add wallet balance constraints: %w", err)
			}

			return nil
		})
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Dropping CHECK constraints from betting_wallet_balances...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `
				ALTER TABLE betting_wallet_balances
				    DROP CONSTRAINT IF EXISTS chk_balance_non_negative,
				    DROP CONSTRAINT IF EXISTS chk_available_non_negative;
			`); err != nil {
				return fmt.Errorf("drop wallet balance constraints: %w", err)
			}

			return nil
		})
	})
}
