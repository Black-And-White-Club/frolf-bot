package usermigrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Print(" [up migration] adding synced_at to club_memberships table...")

		_, err := db.ExecContext(ctx, `
			ALTER TABLE club_memberships ADD COLUMN IF NOT EXISTS synced_at TIMESTAMPTZ;
		`)
		if err != nil {
			return fmt.Errorf("failed to add synced_at to club_memberships: %w", err)
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Print(" [down migration] removing synced_at from club_memberships table...")
		_, err := db.ExecContext(ctx, `
			ALTER TABLE club_memberships DROP COLUMN IF EXISTS synced_at;
		`)
		return err
	})
}
