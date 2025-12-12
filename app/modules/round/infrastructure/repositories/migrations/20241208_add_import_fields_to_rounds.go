package roundmigrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Adding import fields to rounds table...")

		// Note: this migration may run before the rounds table exists (depending on
		// the registration order). Use Postgres' IF EXISTS/IF NOT EXISTS to make it
		// safe in both cases:
		// - Existing DB with rounds table: adds the columns.
		// - Fresh DB without rounds table yet: no-ops (table is created later).
		//
		// This avoids a hard failure like: relation "rounds" does not exist.

		stmts := []string{
			// Backward compatibility: earlier versions used `udisc_url` (no underscore).
			// The canonical DB column name should be `u_disc_url` to match Bun's default
			// snake_case mapping for the Go field name UDiscURL.
			"DO $$\nBEGIN\n  IF EXISTS (\n    SELECT 1\n    FROM information_schema.columns\n    WHERE table_schema = 'public'\n      AND table_name = 'rounds'\n      AND column_name = 'udisc_url'\n  ) AND NOT EXISTS (\n    SELECT 1\n    FROM information_schema.columns\n    WHERE table_schema = 'public'\n      AND table_name = 'rounds'\n      AND column_name = 'u_disc_url'\n  ) THEN\n    ALTER TABLE rounds RENAME COLUMN udisc_url TO u_disc_url;\n  END IF;\nEND\n$$;",
			"ALTER TABLE IF EXISTS rounds ADD COLUMN IF NOT EXISTS import_id TEXT",
			"ALTER TABLE IF EXISTS rounds ADD COLUMN IF NOT EXISTS import_status TEXT",
			"ALTER TABLE IF EXISTS rounds ADD COLUMN IF NOT EXISTS import_type TEXT",
			"ALTER TABLE IF EXISTS rounds ADD COLUMN IF NOT EXISTS file_data BYTEA",
			"ALTER TABLE IF EXISTS rounds ADD COLUMN IF NOT EXISTS file_name TEXT",
			"ALTER TABLE IF EXISTS rounds ADD COLUMN IF NOT EXISTS u_disc_url TEXT",
			"ALTER TABLE IF EXISTS rounds ADD COLUMN IF NOT EXISTS import_user_id TEXT",
			"ALTER TABLE IF EXISTS rounds ADD COLUMN IF NOT EXISTS import_channel_id TEXT",
			"ALTER TABLE IF EXISTS rounds ADD COLUMN IF NOT EXISTS import_notes TEXT",
			"ALTER TABLE IF EXISTS rounds ADD COLUMN IF NOT EXISTS import_error TEXT",
			"ALTER TABLE IF EXISTS rounds ADD COLUMN IF NOT EXISTS import_error_code TEXT",
			"ALTER TABLE IF EXISTS rounds ADD COLUMN IF NOT EXISTS imported_at TIMESTAMP",
		}

		for _, stmt := range stmts {
			if _, err := db.ExecContext(ctx, stmt); err != nil {
				return err
			}
		}

		fmt.Println("Import fields added to rounds table successfully!")
		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Removing import fields from rounds table...")

		stmts := []string{
			"ALTER TABLE IF EXISTS rounds DROP COLUMN IF EXISTS import_id",
			"ALTER TABLE IF EXISTS rounds DROP COLUMN IF EXISTS import_status",
			"ALTER TABLE IF EXISTS rounds DROP COLUMN IF EXISTS import_type",
			"ALTER TABLE IF EXISTS rounds DROP COLUMN IF EXISTS file_data",
			"ALTER TABLE IF EXISTS rounds DROP COLUMN IF EXISTS file_name",
			// Drop both spellings to be resilient across schema versions.
			"ALTER TABLE IF EXISTS rounds DROP COLUMN IF EXISTS u_disc_url",
			"ALTER TABLE IF EXISTS rounds DROP COLUMN IF EXISTS udisc_url",
			"ALTER TABLE IF EXISTS rounds DROP COLUMN IF EXISTS import_user_id",
			"ALTER TABLE IF EXISTS rounds DROP COLUMN IF EXISTS import_channel_id",
			"ALTER TABLE IF EXISTS rounds DROP COLUMN IF EXISTS import_notes",
			"ALTER TABLE IF EXISTS rounds DROP COLUMN IF EXISTS import_error",
			"ALTER TABLE IF EXISTS rounds DROP COLUMN IF EXISTS import_error_code",
			"ALTER TABLE IF EXISTS rounds DROP COLUMN IF EXISTS imported_at",
		}

		for _, stmt := range stmts {
			if _, err := db.ExecContext(ctx, stmt); err != nil {
				return err
			}
		}

		fmt.Println("Import fields removed from rounds table successfully!")
		return nil
	})
}
