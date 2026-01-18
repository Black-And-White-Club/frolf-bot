package usermigrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {

	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Print(" [up migration] adding updated_at to users table...")

		// We only need to add the updated_at column.
		// The unique constraint on memberships already exists from your previous migration.
		_, err := db.ExecContext(ctx, `
			ALTER TABLE users 
			ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;
		`)
		if err != nil {
			return fmt.Errorf("failed to add updated_at to users: %w", err)
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Print(" [down migration] removing updated_at from users table...")
		_, err := db.ExecContext(ctx, `ALTER TABLE users DROP COLUMN IF EXISTS updated_at;`)
		return err
	})
}
