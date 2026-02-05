package clubmigrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Creating clubs table and backfilling from guild_configs...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			// 1. Create clubs table
			if _, err := tx.ExecContext(ctx, `
				CREATE TABLE IF NOT EXISTS clubs (
					uuid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					name VARCHAR(100) NOT NULL DEFAULT 'Disc Golf League',
					icon_url TEXT,
					discord_guild_id VARCHAR(20) UNIQUE,
					created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
					updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
				);
				CREATE INDEX IF NOT EXISTS idx_clubs_discord_guild_id ON clubs(discord_guild_id);
			`); err != nil {
				return fmt.Errorf("failed to create clubs table: %w", err)
			}

			// 2. Backfill from existing guild_configs
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO clubs (uuid, name, discord_guild_id, created_at, updated_at)
				SELECT uuid, 'Disc Golf League', guild_id, created_at, updated_at
				FROM guild_configs
				ON CONFLICT (uuid) DO NOTHING;
			`); err != nil {
				return fmt.Errorf("failed to backfill clubs from guild_configs: %w", err)
			}

			// 3. Update club_memberships FK to reference clubs instead of guild_configs
			// First drop existing FK if it exists
			if _, err := tx.ExecContext(ctx, `
				ALTER TABLE club_memberships
				DROP CONSTRAINT IF EXISTS club_memberships_club_uuid_fkey;
			`); err != nil {
				return fmt.Errorf("failed to drop old FK: %w", err)
			}

			// Add new FK to clubs
			if _, err := tx.ExecContext(ctx, `
				ALTER TABLE club_memberships
				ADD CONSTRAINT fk_club_memberships_club
				FOREIGN KEY (club_uuid) REFERENCES clubs(uuid);
			`); err != nil {
				return fmt.Errorf("failed to add FK to clubs: %w", err)
			}

			return nil
		})
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Rolling back clubs table creation...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			// Restore FK to guild_configs
			if _, err := tx.ExecContext(ctx, `
				ALTER TABLE club_memberships
				DROP CONSTRAINT IF EXISTS fk_club_memberships_club;
			`); err != nil {
				return fmt.Errorf("failed to drop FK: %w", err)
			}

			if _, err := tx.ExecContext(ctx, `
				ALTER TABLE club_memberships
				ADD CONSTRAINT club_memberships_club_uuid_fkey
				FOREIGN KEY (club_uuid) REFERENCES guild_configs(uuid) ON DELETE CASCADE;
			`); err != nil {
				return fmt.Errorf("failed to restore FK: %w", err)
			}

			// Drop clubs table
			if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS clubs;`); err != nil {
				return fmt.Errorf("failed to drop clubs table: %w", err)
			}

			return nil
		})
	})
}
