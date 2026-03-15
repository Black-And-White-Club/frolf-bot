package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Adding source_round_id to betting_wallet_journal...")
		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `
				ALTER TABLE betting_wallet_journal
				ADD COLUMN IF NOT EXISTS source_round_id UUID;
			`); err != nil {
				return fmt.Errorf("add source_round_id column: %w", err)
			}
			if _, err := tx.ExecContext(ctx, `
				CREATE UNIQUE INDEX IF NOT EXISTS idx_betting_wallet_journal_dedup
				ON betting_wallet_journal (club_uuid, user_uuid, season_id, source_round_id)
				WHERE source_round_id IS NOT NULL;
			`); err != nil {
				return fmt.Errorf("create dedup index: %w", err)
			}
			return nil
		})
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Dropping source_round_id from betting_wallet_journal...")
		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `DROP INDEX IF EXISTS idx_betting_wallet_journal_dedup;`); err != nil {
				return fmt.Errorf("drop dedup index: %w", err)
			}
			if _, err := tx.ExecContext(ctx, `ALTER TABLE betting_wallet_journal DROP COLUMN IF EXISTS source_round_id;`); err != nil {
				return fmt.Errorf("drop source_round_id: %w", err)
			}
			return nil
		})
	})
}
