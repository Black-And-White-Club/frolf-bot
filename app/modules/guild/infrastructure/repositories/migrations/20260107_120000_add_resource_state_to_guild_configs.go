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

            // Create enum type first
            _, err := db.ExecContext(ctx, `
                DO $$ 
                BEGIN
                    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'deletion_status_enum') THEN
                        CREATE TYPE deletion_status_enum AS ENUM ('pending', 'completed', 'failed');
                    END IF;
                END $$;
            `)
            if err != nil {
                return fmt.Errorf("failed to create deletion_status_enum: %w", err)
            }

            // Add resource_state column
            _, err = db.ExecContext(ctx, `
                ALTER TABLE guild_configs 
                ADD COLUMN IF NOT EXISTS resource_state jsonb;
            `)
            if err != nil {
                return fmt.Errorf("failed to add resource_state column: %w", err)
            }

            // Add deletion_status column
            _, err = db.ExecContext(ctx, `
                ALTER TABLE guild_configs 
                ADD COLUMN IF NOT EXISTS deletion_status deletion_status_enum;
            `)
            if err != nil {
                return fmt.Errorf("failed to add deletion_status column: %w", err)
            }

            fmt.Println("resource_state and deletion_status columns added successfully")
            return nil
        },
        func(ctx context.Context, db *bun.DB) error {
            fmt.Println("Dropping resource_state and deletion_status from guild_configs...")

            // Drop columns
            _, err := db.ExecContext(ctx, `
                ALTER TABLE guild_configs 
                DROP COLUMN IF EXISTS resource_state,
                DROP COLUMN IF EXISTS deletion_status;
            `)
            if err != nil {
                return fmt.Errorf("failed to drop columns: %w", err)
            }

            // Drop enum type
            _, err = db.ExecContext(ctx, `
                DROP TYPE IF EXISTS deletion_status_enum;
            `)
            if err != nil {
                return fmt.Errorf("failed to drop deletion_status_enum: %w", err)
            }

            fmt.Println("resource_state and deletion_status columns dropped successfully")
            return nil
        },
    )
}
