package usermigrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Print(" [up migration] adding profile fields to users table...")

		_, err := db.ExecContext(ctx, `
			ALTER TABLE users ADD COLUMN IF NOT EXISTS display_name VARCHAR(255);
			ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_hash VARCHAR(255);
			ALTER TABLE users ADD COLUMN IF NOT EXISTS profile_updated_at TIMESTAMPTZ;

			CREATE INDEX IF NOT EXISTS idx_users_display_name ON users(display_name);
		`)
		if err != nil {
			return fmt.Errorf("failed to add profile fields to users: %w", err)
		}

		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Print(" [down migration] removing profile fields from users table...")
		_, err := db.ExecContext(ctx, `
			DROP INDEX IF EXISTS idx_users_display_name;
			ALTER TABLE users DROP COLUMN IF EXISTS display_name;
			ALTER TABLE users DROP COLUMN IF EXISTS avatar_hash;
			ALTER TABLE users DROP COLUMN IF EXISTS profile_updated_at;
		`)
		return err
	})
}
