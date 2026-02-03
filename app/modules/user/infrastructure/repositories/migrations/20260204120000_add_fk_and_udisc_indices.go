package usermigrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Adding FK and UDisc indices for optimization...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			// Foreign key indices
			if _, err := tx.ExecContext(ctx, `
				CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_uuid ON refresh_tokens(user_uuid);
				CREATE INDEX IF NOT EXISTS idx_club_memberships_user_uuid ON club_memberships(user_uuid);
				CREATE INDEX IF NOT EXISTS idx_club_memberships_club_uuid ON club_memberships(club_uuid);
				CREATE INDEX IF NOT EXISTS idx_magic_links_user_uuid ON magic_links(user_uuid);
			`); err != nil {
				return fmt.Errorf("failed to add FK indices: %w", err)
			}

			// Functional indices for case-insensitive lookup
			if _, err := tx.ExecContext(ctx, `
				CREATE INDEX IF NOT EXISTS idx_users_udisc_name_lower ON users(LOWER(udisc_name));
				CREATE INDEX IF NOT EXISTS idx_users_udisc_username_lower ON users(LOWER(udisc_username));
			`); err != nil {
				return fmt.Errorf("failed to add UDisc functional indices: %w", err)
			}

			return nil
		})
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Rolling back FK and UDisc indices...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `
				DROP INDEX IF EXISTS idx_refresh_tokens_user_uuid;
				DROP INDEX IF EXISTS idx_club_memberships_user_uuid;
				DROP INDEX IF EXISTS idx_club_memberships_club_uuid;
				DROP INDEX IF EXISTS idx_magic_links_user_uuid;
				DROP INDEX IF EXISTS idx_users_udisc_name_lower;
				DROP INDEX IF EXISTS idx_users_udisc_username_lower;
			`); err != nil {
				return fmt.Errorf("failed to drop indices: %w", err)
			}
			return nil
		})
	})
}
