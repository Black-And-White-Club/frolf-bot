package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(
		func(ctx context.Context, db *bun.DB) error {
			fmt.Println("Refining guild_configs schema (enum to varchar migration)...")

			// Check if deletion_status column is currently an enum type
			// This handles existing installs that ran the old migration with enum
			var isEnum bool
			err := db.QueryRowContext(ctx, `
				SELECT EXISTS (
					SELECT 1 FROM information_schema.columns c
					JOIN pg_type t ON c.udt_name = t.typname
					WHERE c.table_name = 'guild_configs'
					AND c.column_name = 'deletion_status'
					AND t.typtype = 'e'
				)
			`).Scan(&isEnum)
			if err != nil {
				return fmt.Errorf("failed to check deletion_status column type: %w", err)
			}

			if isEnum {
				fmt.Println("Converting deletion_status from enum to varchar...")

				// Convert enum to varchar, preserving existing values
				_, err = db.ExecContext(ctx, `
					-- Drop any existing constraint first
					ALTER TABLE guild_configs
					DROP CONSTRAINT IF EXISTS deletion_status_check;

					-- Convert column from enum to varchar
					ALTER TABLE guild_configs
					ALTER COLUMN deletion_status TYPE varchar(20)
					USING deletion_status::text;

					-- Set default and NOT NULL
					ALTER TABLE guild_configs
					ALTER COLUMN deletion_status SET DEFAULT 'none';

					ALTER TABLE guild_configs
					ALTER COLUMN deletion_status SET NOT NULL;

					-- Add CHECK constraint
					ALTER TABLE guild_configs
					ADD CONSTRAINT deletion_status_check
					CHECK (deletion_status IN ('none', 'pending', 'completed', 'failed'));

					-- Clean up the old enum type
					DROP TYPE IF EXISTS deletion_status_enum;
				`)
				if err != nil {
					return fmt.Errorf("failed to convert deletion_status to varchar: %w", err)
				}
				fmt.Println("deletion_status converted to varchar successfully")
			} else {
				fmt.Println("deletion_status is already varchar, ensuring constraint exists...")

				// Ensure CHECK constraint exists (idempotent)
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
					return fmt.Errorf("failed to ensure deletion_status check constraint: %w", err)
				}

				// Also drop the enum type if it exists (cleanup)
				_, err = db.ExecContext(ctx, `
					DROP TYPE IF EXISTS deletion_status_enum;
				`)
				if err != nil {
					return fmt.Errorf("failed to drop deletion_status_enum: %w", err)
				}
			}

			// Expand signup_emoji to handle custom Discord emojis like <:frolf:123456789012345678>
			_, err = db.ExecContext(ctx, `
				ALTER TABLE guild_configs
				ALTER COLUMN signup_emoji TYPE varchar(64);
			`)
			if err != nil {
				return fmt.Errorf("failed to expand signup_emoji length: %w", err)
			}

			fmt.Println("guild_configs schema refined successfully")
			return nil
		},
		func(ctx context.Context, db *bun.DB) error {
			fmt.Println("Rolling back guild_configs schema refinement...")

			// Note: We cannot easily recreate the enum from varchar
			// This rollback just removes the constraint and shrinks emoji column
			_, err := db.ExecContext(ctx, `
				ALTER TABLE guild_configs
				DROP CONSTRAINT IF EXISTS deletion_status_check;

				ALTER TABLE guild_configs
				ALTER COLUMN signup_emoji TYPE varchar(10);
			`)
			if err != nil {
				return fmt.Errorf("failed to rollback schema changes: %w", err)
			}

			fmt.Println("guild_configs schema rollback completed")
			return nil
		},
	)
}
