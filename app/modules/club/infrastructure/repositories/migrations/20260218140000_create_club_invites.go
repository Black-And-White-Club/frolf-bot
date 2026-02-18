package clubmigrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Creating club_invites table...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `
				CREATE TABLE IF NOT EXISTS club_invites (
					uuid          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					club_uuid     UUID NOT NULL REFERENCES clubs(uuid) ON DELETE CASCADE,
					created_by    UUID NOT NULL REFERENCES users(uuid) ON DELETE CASCADE,
					code          VARCHAR(12) UNIQUE NOT NULL,
					role          TEXT NOT NULL DEFAULT 'player',
					max_uses      INT DEFAULT NULL,
					use_count     INT NOT NULL DEFAULT 0,
					expires_at    TIMESTAMPTZ DEFAULT NULL,
					revoked       BOOL NOT NULL DEFAULT FALSE,
					created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
				);
				CREATE INDEX IF NOT EXISTS idx_club_invites_club_uuid ON club_invites(club_uuid);
				CREATE INDEX IF NOT EXISTS idx_club_invites_code ON club_invites(code);
			`); err != nil {
				return fmt.Errorf("failed to create club_invites table: %w", err)
			}
			return nil
		})
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Rolling back club_invites table...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS club_invites;`); err != nil {
				return fmt.Errorf("failed to drop club_invites: %w", err)
			}
			return nil
		})
	})
}
