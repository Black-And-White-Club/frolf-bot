package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Adding betting wallet balances table...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `
				CREATE TABLE IF NOT EXISTS betting_wallet_balances (
					club_uuid UUID NOT NULL,
					user_uuid UUID NOT NULL,
					season_id VARCHAR(64) NOT NULL,
					balance   INTEGER NOT NULL DEFAULT 0,
					reserved  INTEGER NOT NULL DEFAULT 0,
					PRIMARY KEY (club_uuid, user_uuid, season_id),
					CHECK (reserved >= 0)
				);
			`); err != nil {
				return fmt.Errorf("create betting_wallet_balances: %w", err)
			}

			return nil
		})
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Dropping betting wallet balances table...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS betting_wallet_balances;`); err != nil {
				return fmt.Errorf("drop betting_wallet_balances: %w", err)
			}
			return nil
		})
	})
}
