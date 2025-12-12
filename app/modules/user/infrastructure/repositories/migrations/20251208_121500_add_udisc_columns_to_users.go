package usermigrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Adding UDisc columns to users table...")

		// NOTE: The users table is created from the current bun model in an earlier migration.
		// That means fresh databases may already include these columns.
		// Use IF NOT EXISTS to keep migrations forward-compatible and safe to re-run.
		if _, err := db.ExecContext(ctx, "ALTER TABLE users ADD COLUMN IF NOT EXISTS udisc_username TEXT NULL"); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, "ALTER TABLE users ADD COLUMN IF NOT EXISTS udisc_display_name TEXT NULL"); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, "ALTER TABLE users ADD COLUMN IF NOT EXISTS normalized_username TEXT NOT NULL DEFAULT ''"); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, "ALTER TABLE users ADD COLUMN IF NOT EXISTS normalized_display_name TEXT NOT NULL DEFAULT ''"); err != nil {
			return err
		}

		fmt.Println("UDisc columns added successfully!")
		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Dropping UDisc columns from users table...")

		if _, err := db.ExecContext(ctx, "ALTER TABLE users DROP COLUMN IF EXISTS udisc_username"); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, "ALTER TABLE users DROP COLUMN IF EXISTS udisc_display_name"); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, "ALTER TABLE users DROP COLUMN IF EXISTS normalized_username"); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, "ALTER TABLE users DROP COLUMN IF EXISTS normalized_display_name"); err != nil {
			return err
		}

		fmt.Println("UDisc columns dropped successfully!")
		return nil
	})
}
