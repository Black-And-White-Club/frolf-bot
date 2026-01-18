package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(
		func(ctx context.Context, db *bun.DB) error {
			fmt.Println("Adding resource_state and deletion_status to guild_configs...")

			// Add resource_state column (JSONB for flexible resource tracking)
			_, err := db.ExecContext(ctx, `
				ALTER TABLE guild_configs
				ADD COLUMN IF NOT EXISTS resource_state jsonb;
			`)
			if err != nil {
				return fmt.Errorf("failed to add resource_state column: %w", err)
			}

			// Add deletion_status as VARCHAR with CHECK constraint
			// This is more maintainable than PostgreSQL enums for evolving schemas
			_, err = db.ExecContext(ctx, `
				ALTER TABLE guild_configs
				ADD COLUMN IF NOT EXISTS deletion_status varchar(20) DEFAULT 'none' NOT NULL;
			`)
			if err != nil {
				return fmt.Errorf("failed to add deletion_status column: %w", err)
			}

			// Add CHECK constraint for valid values
			_, err = db.ExecContext(ctx, `
				DO $$
				BEGIN
					IF NOT EXISTS (
						SELECT 1 FROM pg_constraint
						WHERE conname = 'deletion_status_check'
					) THEN
						ALTER TABLE guild_configs
						ADD CONSTRAINT deletion_status_check
						CHECK (deletion_status IN ('none', 'pending', 'completed', 'failed'));
					END IF;
				END $$;
			`)
			if err != nil {
				return fmt.Errorf("failed to add deletion_status check constraint: %w", err)
			}

			fmt.Println("resource_state and deletion_status columns added successfully")
			return nil
		},
		func(ctx context.Context, db *bun.DB) error {
			fmt.Println("Dropping resource_state and deletion_status from guild_configs...")

			// Drop constraint first
			_, err := db.ExecContext(ctx, `
				ALTER TABLE guild_configs
				DROP CONSTRAINT IF EXISTS deletion_status_check;
			`)
			if err != nil {
				return fmt.Errorf("failed to drop deletion_status_check constraint: %w", err)
			}

			// Drop columns
			_, err = db.ExecContext(ctx, `
				ALTER TABLE guild_configs
				DROP COLUMN IF EXISTS resource_state,
				DROP COLUMN IF EXISTS deletion_status;
			`)
			if err != nil {
				return fmt.Errorf("failed to drop columns: %w", err)
			}

			fmt.Println("resource_state and deletion_status columns dropped successfully")
			return nil
		},
	)
}
