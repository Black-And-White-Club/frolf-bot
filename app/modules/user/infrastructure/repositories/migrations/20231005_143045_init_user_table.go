package usermigrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	// Ensure migration caller discovery is enabled so this migration gets a stable ID
	// even if init file ordering is not deterministic.
	if err := Migrations.DiscoverCaller(); err != nil {
		panic(err)
	}
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Creating users table (monolithic schema)...")

		// Freeze the schema exactly as it was when first deployed to Prod
		_, err := db.ExecContext(ctx, `
			CREATE TABLE IF NOT EXISTS users (
				id BIGSERIAL PRIMARY KEY,
				user_id TEXT NOT NULL,
				guild_id TEXT NOT NULL,
				role TEXT NOT NULL DEFAULT 'User',
				created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				UNIQUE(user_id, guild_id)
			);
		`)
		if err != nil {
			return fmt.Errorf("failed to create users table: %w", err)
		}

		fmt.Println("Users table created successfully!")
		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Dropping users table...")

		_, err := db.ExecContext(ctx, `DROP TABLE IF EXISTS users;`)
		if err != nil {
			return fmt.Errorf("failed to drop users table: %w", err)
		}

		fmt.Println("Users table dropped successfully!")
		return nil
	})
}
