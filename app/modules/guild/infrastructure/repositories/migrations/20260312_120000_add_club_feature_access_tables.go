package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Creating club feature access tables...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `
				CREATE TABLE IF NOT EXISTS club_feature_overrides (
					club_uuid UUID NOT NULL,
					feature_key VARCHAR(64) NOT NULL,
					state VARCHAR(20) NOT NULL,
					reason TEXT NOT NULL DEFAULT '',
					expires_at TIMESTAMPTZ,
					updated_by VARCHAR(128) NOT NULL DEFAULT '',
					created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
					PRIMARY KEY (club_uuid, feature_key),
					CONSTRAINT club_feature_overrides_state_check
						CHECK (state IN ('enabled', 'disabled', 'frozen'))
				);
			`); err != nil {
				return fmt.Errorf("create club_feature_overrides: %w", err)
			}

			if _, err := tx.ExecContext(ctx, `
				CREATE INDEX IF NOT EXISTS idx_club_feature_overrides_feature
				ON club_feature_overrides (feature_key);
			`); err != nil {
				return fmt.Errorf("create idx_club_feature_overrides_feature: %w", err)
			}

			if _, err := tx.ExecContext(ctx, `
				CREATE TABLE IF NOT EXISTS club_feature_access_audit (
					id BIGSERIAL PRIMARY KEY,
					club_uuid UUID NOT NULL,
					guild_id VARCHAR(20) NOT NULL,
					feature_key VARCHAR(64) NOT NULL,
					state VARCHAR(20) NOT NULL,
					source VARCHAR(32) NOT NULL,
					reason TEXT NOT NULL DEFAULT '',
					updated_by VARCHAR(128) NOT NULL DEFAULT '',
					expires_at TIMESTAMPTZ,
					created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
					CONSTRAINT club_feature_access_audit_state_check
						CHECK (state IN ('enabled', 'disabled', 'frozen')),
					CONSTRAINT club_feature_access_audit_source_check
						CHECK (source IN ('none', 'subscription', 'trial', 'manual_allow', 'manual_deny'))
				);
			`); err != nil {
				return fmt.Errorf("create club_feature_access_audit: %w", err)
			}

			if _, err := tx.ExecContext(ctx, `
				CREATE INDEX IF NOT EXISTS idx_club_feature_access_audit_club_feature
				ON club_feature_access_audit (club_uuid, feature_key, created_at DESC);
			`); err != nil {
				return fmt.Errorf("create idx_club_feature_access_audit_club_feature: %w", err)
			}

			return nil
		})
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Dropping club feature access tables...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS club_feature_access_audit;`); err != nil {
				return fmt.Errorf("drop club_feature_access_audit: %w", err)
			}
			if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS club_feature_overrides;`); err != nil {
				return fmt.Errorf("drop club_feature_overrides: %w", err)
			}
			return nil
		})
	})
}
