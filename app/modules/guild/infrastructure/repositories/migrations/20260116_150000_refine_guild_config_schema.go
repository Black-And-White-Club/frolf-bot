package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	// Ensure the name here matches your filename (e.g., 20260118_refine_guild_configs)
	Migrations.MustRegister(
		func(ctx context.Context, db *bun.DB) error {
			fmt.Println("Executing robust refinement of guild_configs...")

			_, err := db.ExecContext(ctx, `
				-- 1. Unlock the column by dropping defaults/constraints
				ALTER TABLE guild_configs ALTER COLUMN deletion_status DROP DEFAULT;
				ALTER TABLE guild_configs DROP CONSTRAINT IF EXISTS deletion_status_check;

				-- 2. Force conversion to varchar(20)
				-- We cast to text first to break the enum bond
				ALTER TABLE guild_configs 
				ALTER COLUMN deletion_status TYPE varchar(20) 
				USING deletion_status::text;

				-- 3. Set standard defaults and constraints
				ALTER TABLE guild_configs ALTER COLUMN deletion_status SET DEFAULT 'none';
				UPDATE guild_configs SET deletion_status = 'none' WHERE deletion_status IS NULL;
				ALTER TABLE guild_configs ALTER COLUMN deletion_status SET NOT NULL;

				ALTER TABLE guild_configs ADD CONSTRAINT deletion_status_check 
					CHECK (deletion_status IN ('none', 'pending', 'completed', 'failed'));

				-- 4. Fix signup_emoji length (important for custom Discord emojis)
				ALTER TABLE guild_configs ALTER COLUMN signup_emoji TYPE varchar(64);

				-- 5. Final cleanup of the old type
				DROP TYPE IF EXISTS deletion_status_enum CASCADE;
			`)
			if err != nil {
				return fmt.Errorf("migration failed: %w", err)
			}

			fmt.Println("guild_configs refined successfully")
			return nil
		},
		func(ctx context.Context, db *bun.DB) error {
			// Rollback: remove constraint and shrink emoji column
			_, err := db.ExecContext(ctx, `
				ALTER TABLE guild_configs DROP CONSTRAINT IF EXISTS deletion_status_check;
				ALTER TABLE guild_configs ALTER COLUMN signup_emoji TYPE varchar(10);
			`)
			return err
		},
	)
}
