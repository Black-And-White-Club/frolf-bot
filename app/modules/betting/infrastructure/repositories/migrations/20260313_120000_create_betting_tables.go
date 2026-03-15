package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Creating betting tables...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `
				CREATE TABLE IF NOT EXISTS betting_member_settings (
					club_uuid UUID NOT NULL,
					user_uuid UUID NOT NULL,
					opt_out_targeting BOOLEAN NOT NULL DEFAULT FALSE,
					updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
					PRIMARY KEY (club_uuid, user_uuid)
				);
			`); err != nil {
				return fmt.Errorf("create betting_member_settings: %w", err)
			}

			if _, err := tx.ExecContext(ctx, `
				CREATE TABLE IF NOT EXISTS betting_wallet_journal (
					id BIGSERIAL PRIMARY KEY,
					club_uuid UUID NOT NULL,
					user_uuid UUID NOT NULL,
					season_id VARCHAR(64) NOT NULL,
					entry_type VARCHAR(64) NOT NULL,
					amount INTEGER NOT NULL,
					reason TEXT NOT NULL DEFAULT '',
					created_by VARCHAR(128) NOT NULL DEFAULT '',
					created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
				);
			`); err != nil {
				return fmt.Errorf("create betting_wallet_journal: %w", err)
			}

			if _, err := tx.ExecContext(ctx, `
				CREATE INDEX IF NOT EXISTS idx_betting_wallet_journal_wallet
				ON betting_wallet_journal (club_uuid, user_uuid, season_id, created_at DESC);
			`); err != nil {
				return fmt.Errorf("create idx_betting_wallet_journal_wallet: %w", err)
			}

			return nil
		})
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Dropping betting tables...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS betting_wallet_journal;`); err != nil {
				return fmt.Errorf("drop betting_wallet_journal: %w", err)
			}
			if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS betting_member_settings;`); err != nil {
				return fmt.Errorf("drop betting_member_settings: %w", err)
			}
			return nil
		})
	})
}
