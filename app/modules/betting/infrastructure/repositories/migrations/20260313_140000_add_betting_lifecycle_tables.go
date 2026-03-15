package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Adding betting lifecycle tables...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			statements := []string{
				`ALTER TABLE betting_markets ADD COLUMN IF NOT EXISTS resolved_option_key TEXT NOT NULL DEFAULT '';`,
				`ALTER TABLE betting_markets ADD COLUMN IF NOT EXISTS void_reason TEXT NOT NULL DEFAULT '';`,
				`ALTER TABLE betting_markets ADD COLUMN IF NOT EXISTS result_summary TEXT NOT NULL DEFAULT '';`,
				`ALTER TABLE betting_markets ADD COLUMN IF NOT EXISTS settlement_version INTEGER NOT NULL DEFAULT 0;`,
				`ALTER TABLE betting_markets ADD COLUMN IF NOT EXISTS last_result_source VARCHAR(128) NOT NULL DEFAULT '';`,
				`ALTER TABLE betting_markets ADD COLUMN IF NOT EXISTS settled_at TIMESTAMPTZ;`,
				`ALTER TABLE betting_bets ADD COLUMN IF NOT EXISTS settled_payout INTEGER NOT NULL DEFAULT 0;`,
				`ALTER TABLE betting_bets ADD COLUMN IF NOT EXISTS settled_at TIMESTAMPTZ;`,
				`CREATE INDEX IF NOT EXISTS idx_betting_markets_round_lookup ON betting_markets (club_uuid, round_id);`,
				`CREATE INDEX IF NOT EXISTS idx_betting_bets_market ON betting_bets (market_id, status, created_at DESC);`,
				`
				CREATE TABLE IF NOT EXISTS betting_audit_log (
					id BIGSERIAL PRIMARY KEY,
					club_uuid UUID NOT NULL,
					market_id BIGINT NULL REFERENCES betting_markets(id) ON DELETE SET NULL,
					round_id UUID NULL,
					actor_user_uuid UUID NULL,
					action VARCHAR(64) NOT NULL,
					reason TEXT NOT NULL DEFAULT '',
					metadata TEXT NOT NULL DEFAULT '',
					created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
				);
				`,
				`CREATE INDEX IF NOT EXISTS idx_betting_audit_log_market ON betting_audit_log (club_uuid, market_id, created_at DESC);`,
			}

			for _, stmt := range statements {
				if _, err := tx.ExecContext(ctx, stmt); err != nil {
					return fmt.Errorf("apply betting lifecycle statement: %w", err)
				}
			}

			return nil
		})
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Removing betting lifecycle tables...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			statements := []string{
				`DROP INDEX IF EXISTS idx_betting_audit_log_market;`,
				`DROP TABLE IF EXISTS betting_audit_log;`,
				`DROP INDEX IF EXISTS idx_betting_bets_market;`,
				`DROP INDEX IF EXISTS idx_betting_markets_round_lookup;`,
				`ALTER TABLE betting_bets DROP COLUMN IF EXISTS settled_at;`,
				`ALTER TABLE betting_bets DROP COLUMN IF EXISTS settled_payout;`,
				`ALTER TABLE betting_markets DROP COLUMN IF EXISTS settled_at;`,
				`ALTER TABLE betting_markets DROP COLUMN IF EXISTS last_result_source;`,
				`ALTER TABLE betting_markets DROP COLUMN IF EXISTS settlement_version;`,
				`ALTER TABLE betting_markets DROP COLUMN IF EXISTS result_summary;`,
				`ALTER TABLE betting_markets DROP COLUMN IF EXISTS void_reason;`,
				`ALTER TABLE betting_markets DROP COLUMN IF EXISTS resolved_option_key;`,
			}

			for _, stmt := range statements {
				if _, err := tx.ExecContext(ctx, stmt); err != nil {
					return fmt.Errorf("rollback betting lifecycle statement: %w", err)
				}
			}

			return nil
		})
	})
}
