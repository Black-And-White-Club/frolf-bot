package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Adding betting market tables...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `
				CREATE TABLE IF NOT EXISTS betting_markets (
					id BIGSERIAL PRIMARY KEY,
					club_uuid UUID NOT NULL,
					season_id VARCHAR(64) NOT NULL,
					round_id UUID NOT NULL,
					market_type VARCHAR(64) NOT NULL,
					title TEXT NOT NULL,
					status VARCHAR(32) NOT NULL,
					locks_at TIMESTAMPTZ NOT NULL,
					created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
					UNIQUE (club_uuid, season_id, round_id, market_type)
				);
			`); err != nil {
				return fmt.Errorf("create betting_markets: %w", err)
			}

			if _, err := tx.ExecContext(ctx, `
				CREATE TABLE IF NOT EXISTS betting_market_options (
					id BIGSERIAL PRIMARY KEY,
					market_id BIGINT NOT NULL REFERENCES betting_markets(id) ON DELETE CASCADE,
					option_key VARCHAR(128) NOT NULL,
					participant_member_id VARCHAR(32) NOT NULL,
					label TEXT NOT NULL,
					probability_bps INTEGER NOT NULL,
					decimal_odds_cents INTEGER NOT NULL,
					display_order INTEGER NOT NULL,
					UNIQUE (market_id, option_key)
				);
			`); err != nil {
				return fmt.Errorf("create betting_market_options: %w", err)
			}

			if _, err := tx.ExecContext(ctx, `
				CREATE TABLE IF NOT EXISTS betting_bets (
					id BIGSERIAL PRIMARY KEY,
					club_uuid UUID NOT NULL,
					user_uuid UUID NOT NULL,
					season_id VARCHAR(64) NOT NULL,
					round_id UUID NOT NULL,
					market_id BIGINT NOT NULL REFERENCES betting_markets(id) ON DELETE CASCADE,
					market_type VARCHAR(64) NOT NULL,
					selection_key VARCHAR(128) NOT NULL,
					selection_label TEXT NOT NULL,
					stake INTEGER NOT NULL,
					decimal_odds_cents INTEGER NOT NULL,
					potential_payout INTEGER NOT NULL,
					status VARCHAR(32) NOT NULL,
					created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
				);
			`); err != nil {
				return fmt.Errorf("create betting_bets: %w", err)
			}

			if _, err := tx.ExecContext(ctx, `
				CREATE INDEX IF NOT EXISTS idx_betting_markets_round
				ON betting_markets (club_uuid, season_id, round_id, market_type);
			`); err != nil {
				return fmt.Errorf("create idx_betting_markets_round: %w", err)
			}

			if _, err := tx.ExecContext(ctx, `
				CREATE INDEX IF NOT EXISTS idx_betting_bets_wallet
				ON betting_bets (club_uuid, user_uuid, season_id, status, created_at DESC);
			`); err != nil {
				return fmt.Errorf("create idx_betting_bets_wallet: %w", err)
			}

			return nil
		})
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Dropping betting market tables...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS betting_bets;`); err != nil {
				return fmt.Errorf("drop betting_bets: %w", err)
			}
			if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS betting_market_options;`); err != nil {
				return fmt.Errorf("drop betting_market_options: %w", err)
			}
			if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS betting_markets;`); err != nil {
				return fmt.Errorf("drop betting_markets: %w", err)
			}
			return nil
		})
	})
}
