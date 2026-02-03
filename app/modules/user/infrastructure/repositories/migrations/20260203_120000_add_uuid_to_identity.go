package usermigrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Adding UUID columns to users and guild_configs...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			// Ensure pgcrypto is enabled for gen_random_uuid()
			if _, err := tx.ExecContext(ctx, "CREATE EXTENSION IF NOT EXISTS pgcrypto;"); err != nil {
				return fmt.Errorf("failed to enable pgcrypto: %w", err)
			}

			// Add uuid to users
			if _, err := tx.ExecContext(ctx, `
				ALTER TABLE users ADD COLUMN IF NOT EXISTS uuid UUID UNIQUE NOT NULL DEFAULT gen_random_uuid();
			`); err != nil {
				return fmt.Errorf("failed to add uuid to users: %w", err)
			}

			// Add uuid to guild_configs
			if _, err := tx.ExecContext(ctx, `
				ALTER TABLE guild_configs ADD COLUMN IF NOT EXISTS uuid UUID UNIQUE NOT NULL DEFAULT gen_random_uuid();
			`); err != nil {
				return fmt.Errorf("failed to add uuid to guild_configs: %w", err)
			}

			// Add composite index on (guild_id, user_id) in guild_memberships
			if _, err := tx.ExecContext(ctx, `
				CREATE INDEX IF NOT EXISTS idx_guild_memberships_guild_user_id ON guild_memberships(guild_id, user_id);
			`); err != nil {
				return fmt.Errorf("failed to create composite index on guild_memberships: %w", err)
			}

			return nil
		})
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Rolling back UUID columns...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `DROP INDEX IF EXISTS idx_guild_memberships_guild_user_id;`); err != nil {
				return fmt.Errorf("failed to drop index: %w", err)
			}
			if _, err := tx.ExecContext(ctx, `ALTER TABLE guild_configs DROP COLUMN IF EXISTS uuid;`); err != nil {
				return fmt.Errorf("failed to drop uuid from guild_configs: %w", err)
			}
			if _, err := tx.ExecContext(ctx, `ALTER TABLE users DROP COLUMN IF EXISTS uuid;`); err != nil {
				return fmt.Errorf("failed to drop uuid from users: %w", err)
			}
			return nil
		})
	})
}
