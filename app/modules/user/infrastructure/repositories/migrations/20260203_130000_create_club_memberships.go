package usermigrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Print(" [up] creating club_memberships table")

		_, err := db.ExecContext(ctx, `
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
		`)
		return err
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Print(" [down] dropping club_memberships table")
		_, err := db.ExecContext(ctx, `DROP TABLE IF EXISTS club_memberships;`)
		return err
	})
}
