package usermigrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Print(" [up] backfilling club_memberships from guild_memberships")

		_, err := db.ExecContext(ctx, `
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
		`)
		return err
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Print(" [down] cleaning up club_memberships backfill")
		// During rollback, we might not want to delete everything if new data was added,
		// but since this is a transition table, dropping it in the previous migration's down is enough.
		// Here we just do nothing or delete what we inserted if we can track it.
		// For simplicity, we'll let the drop table migration handle it.
		return nil
	})
}
