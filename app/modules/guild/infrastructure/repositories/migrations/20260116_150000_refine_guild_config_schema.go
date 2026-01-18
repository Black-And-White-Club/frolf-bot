package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(
		func(ctx context.Context, db *bun.DB) error {
			fmt.Println("Refining deletion_status_enum and adding defaults...")

			// 1. Add 'none' to the enum type (Postgres requires separate ALTER for each value)
			// We use a transaction-safe check
			_, err := db.ExecContext(ctx, `
                ALTER TYPE deletion_status_enum ADD VALUE IF NOT EXISTS 'none' BEFORE 'pending';
            `)
			if err != nil {
				return fmt.Errorf("failed to add 'none' to enum: %w", err)
			}

			// 2. Set existing NULLs to 'none' and apply constraints
			_, err = db.ExecContext(ctx, `
                UPDATE guild_configs SET deletion_status = 'none' WHERE deletion_status IS NULL;
                ALTER TABLE guild_configs ALTER COLUMN deletion_status SET DEFAULT 'none';
                ALTER TABLE guild_configs ALTER COLUMN deletion_status SET NOT NULL;
            `)
			if err != nil {
				return fmt.Errorf("failed to set deletion_status defaults: %w", err)
			}

			// 3. Update Emoji length to handle custom Discord emojis
			_, err = db.ExecContext(ctx, `
                ALTER TABLE guild_configs ALTER COLUMN signup_emoji TYPE varchar(64);
            `)
			if err != nil {
				return fmt.Errorf("failed to expand signup_emoji length: %w", err)
			}

			fmt.Println("guild_configs schema refined successfully")
			return nil
		},
		func(ctx context.Context, db *bun.DB) error {
			// Rollback: remove NOT NULL and default (we can't easily remove an enum value in PG)
			_, err := db.ExecContext(ctx, `
                ALTER TABLE guild_configs ALTER COLUMN deletion_status DROP NOT NULL;
                ALTER TABLE guild_configs ALTER COLUMN deletion_status DROP DEFAULT;
                ALTER TABLE guild_configs ALTER COLUMN signup_emoji TYPE varchar(10);
            `)
			return err
		},
	)
}
