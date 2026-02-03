package usermigrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Setting up club memberships (UUIDs, Table, Backfill)...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			// 1. Add UUIDs to users
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

			// Add composite index on (guild_id, user_id) in guild_memberships (helper for joins)
			if _, err := tx.ExecContext(ctx, `
				CREATE INDEX IF NOT EXISTS idx_guild_memberships_guild_user_id ON guild_memberships(guild_id, user_id);
			`); err != nil {
				return fmt.Errorf("failed to create composite index on guild_memberships: %w", err)
			}

			// 2. Create club_memberships table
			if _, err := tx.ExecContext(ctx, `
				CREATE TABLE IF NOT EXISTS club_memberships (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					user_uuid UUID NOT NULL REFERENCES users(uuid) ON DELETE CASCADE,
					club_uuid UUID NOT NULL REFERENCES guild_configs(uuid) ON DELETE CASCADE,
					display_name VARCHAR(255),
					avatar_url TEXT,
					role VARCHAR(50) NOT NULL DEFAULT 'user',
					source VARCHAR(50) NOT NULL DEFAULT 'discord',
					external_id VARCHAR(255),
					joined_at TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
					updated_at TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
					UNIQUE (user_uuid, club_uuid)
				);
				CREATE INDEX IF NOT EXISTS idx_club_memberships_user_uuid ON club_memberships(user_uuid);
				CREATE INDEX IF NOT EXISTS idx_club_memberships_club_uuid ON club_memberships(club_uuid);
				CREATE INDEX IF NOT EXISTS idx_club_memberships_updated_at ON club_memberships(updated_at);
				CREATE INDEX IF NOT EXISTS idx_club_memberships_external_id ON club_memberships(external_id);
			`); err != nil {
				return fmt.Errorf("failed to create club_memberships table: %w", err)
			}

			// 3. Backfill club_memberships
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO club_memberships (user_uuid, club_uuid, display_name, role, source, external_id, joined_at)
				SELECT 
					u.uuid,
					gc.uuid,
					u.display_name,
					gm.role,
					'discord',
					gm.user_id,
					gm.joined_at
				FROM guild_memberships gm
				JOIN users u ON u.user_id = gm.user_id
				JOIN guild_configs gc ON gc.guild_id = gm.guild_id
				ON CONFLICT (user_uuid, club_uuid) DO NOTHING;
			`); err != nil {
				return fmt.Errorf("failed to backfill club_memberships: %w", err)
			}

			return nil
		})
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Rolling back club memberships setup...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			// Drop table
			if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS club_memberships;`); err != nil {
				return fmt.Errorf("failed to drop club_memberships: %w", err)
			}

			// Drop index
			if _, err := tx.ExecContext(ctx, `DROP INDEX IF EXISTS idx_guild_memberships_guild_user_id;`); err != nil {
				return fmt.Errorf("failed to drop index: %w", err)
			}

			// Drop UUID column from users
			if _, err := tx.ExecContext(ctx, `ALTER TABLE users DROP COLUMN IF EXISTS uuid;`); err != nil {
				return fmt.Errorf("failed to drop uuid from users: %w", err)
			}

			return nil
		})
	})
}
