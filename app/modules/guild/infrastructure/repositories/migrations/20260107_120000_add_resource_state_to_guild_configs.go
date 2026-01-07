package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
    Migrations.MustRegister(
        func(ctx context.Context, db *bun.DB) error {
            fmt.Println("Adding resource_state column to guild_configs...")
            _, err := db.ExecContext(ctx, `ALTER TABLE guild_configs ADD COLUMN IF NOT EXISTS resource_state jsonb;`)
            if err != nil {
                return fmt.Errorf("failed to add resource_state column: %w", err)
            }
            fmt.Println("resource_state column added successfully")
            return nil
        },
        func(ctx context.Context, db *bun.DB) error {
            fmt.Println("Dropping resource_state column from guild_configs...")
            _, err := db.ExecContext(ctx, `ALTER TABLE guild_configs DROP COLUMN IF EXISTS resource_state;`)
            if err != nil {
                return fmt.Errorf("failed to drop resource_state column: %w", err)
            }
            fmt.Println("resource_state column dropped successfully")
            return nil
        },
    )
}
