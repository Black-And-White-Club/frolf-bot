package usermigrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {

	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Splitting users table into users + guild_memberships...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			// Step 1: Create new users table (global identity)
			if _, err := tx.ExecContext(ctx, `
				CREATE TABLE IF NOT EXISTS users_new (
					id BIGSERIAL PRIMARY KEY,
					user_id TEXT UNIQUE NOT NULL,
						udisc_username TEXT,
						udisc_name TEXT,
						created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
						-- Keep updated_at on the identity table so newer code can rely on it.
						updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
				);
			`); err != nil {
				return fmt.Errorf("failed to create users_new table: %w", err)
			}

			// Step 2: Create guild_memberships table (Logical FK to Guild)
			// guild_id is stored but not constrained to guild_configs to enable order-independent module migrations
			if _, err := tx.ExecContext(ctx, `
				CREATE TABLE IF NOT EXISTS guild_memberships (
					id BIGSERIAL PRIMARY KEY,
					user_id TEXT NOT NULL,
					guild_id TEXT NOT NULL,
					role TEXT NOT NULL DEFAULT 'User',
					joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
					UNIQUE(user_id, guild_id),
					CONSTRAINT fk_guild_memberships_user FOREIGN KEY (user_id) REFERENCES users_new(user_id) ON DELETE CASCADE
				);
			`); err != nil {
				return fmt.Errorf("failed to create guild_memberships table: %w", err)
			}

			// Step 3: Migrate unique users to identity table (deduplicate by user_id)
			// Use a conditional block so this migration works whether or not the
			// original `users` table already has an `updated_at` column.
			if _, err := tx.ExecContext(ctx, `
				DO $$
				BEGIN
					IF EXISTS (
						SELECT 1 FROM information_schema.columns
						WHERE table_name = 'users' AND column_name = 'updated_at'
					) THEN
						INSERT INTO users_new (user_id, udisc_username, udisc_name, created_at, updated_at)
						SELECT DISTINCT ON (user_id) user_id, udisc_username, udisc_name, created_at, COALESCE(updated_at, created_at)
						FROM users
						ORDER BY user_id, id;
					ELSE
						INSERT INTO users_new (user_id, udisc_username, udisc_name, created_at, updated_at)
						SELECT DISTINCT ON (user_id) user_id, udisc_username, udisc_name, created_at, created_at
						FROM users
						ORDER BY user_id, id;
					END IF;
				END $$;
			`); err != nil {
				return fmt.Errorf("failed to migrate users: %w", err)
			}

			// Step 4: Migrate guild memberships (one row per user per guild)
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO guild_memberships (user_id, guild_id, role, joined_at)
				SELECT user_id, guild_id, role, created_at FROM users;
			`); err != nil {
				return fmt.Errorf("failed to migrate guild memberships: %w", err)
			}

			// Step 5: Drop old users table
			if _, err := tx.ExecContext(ctx, `DROP TABLE users;`); err != nil {
				return fmt.Errorf("failed to drop old users table: %w", err)
			}

			// Step 6: Rename users_new to users
			if _, err := tx.ExecContext(ctx, `ALTER TABLE users_new RENAME TO users;`); err != nil {
				return fmt.Errorf("failed to rename users_new: %w", err)
			}

			// Step 7: Create indexes
			if _, err := tx.ExecContext(ctx, `
				CREATE INDEX IF NOT EXISTS idx_guild_memberships_user_id ON guild_memberships(user_id);
				CREATE INDEX IF NOT EXISTS idx_guild_memberships_guild_id ON guild_memberships(guild_id);
			`); err != nil {
				return fmt.Errorf("failed to create indexes: %w", err)
			}

			fmt.Println("Migration completed successfully!")
			return nil
		})
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Rolling back users/guild_memberships split...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			// Rollback: Recreate old users table with guild_id
			if _, err := tx.ExecContext(ctx, `
				CREATE TABLE users_old (
					id BIGSERIAL PRIMARY KEY,
					user_id TEXT NOT NULL,
					guild_id TEXT NOT NULL,
					role TEXT NOT NULL DEFAULT 'User',
					udisc_username TEXT,
					udisc_name TEXT,
					created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
					updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
					UNIQUE(user_id, guild_id)
				);
			`); err != nil {
				return fmt.Errorf("failed to create users_old table: %w", err)
			}

			// Migrate data back: join users and guild_memberships
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO users_old (user_id, guild_id, role, udisc_username, udisc_name, created_at, updated_at)
				SELECT u.user_id, gm.guild_id, gm.role, u.udisc_username, u.udisc_name, u.created_at, COALESCE(u.updated_at, u.created_at)
				FROM users u
				JOIN guild_memberships gm ON u.user_id = gm.user_id;
			`); err != nil {
				return fmt.Errorf("failed to migrate data back: %w", err)
			}

			// Drop new tables
			if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS guild_memberships;`); err != nil {
				return fmt.Errorf("failed to drop guild_memberships: %w", err)
			}
			if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS users;`); err != nil {
				return fmt.Errorf("failed to drop users: %w", err)
			}

			// Rename back
			if _, err := tx.ExecContext(ctx, `ALTER TABLE users_old RENAME TO users;`); err != nil {
				return fmt.Errorf("failed to restore users table: %w", err)
			}

			fmt.Println("Rollback completed!")
			return nil
		})
	})
}
