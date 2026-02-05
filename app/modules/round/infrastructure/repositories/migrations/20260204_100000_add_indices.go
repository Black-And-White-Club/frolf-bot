package roundmigrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Adding missing indices for rounds and round_groups...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `
				CREATE INDEX IF NOT EXISTS idx_round_groups_round_id ON round_groups(round_id);
				CREATE INDEX IF NOT EXISTS idx_round_group_members_group_id ON round_group_members(group_id);
				CREATE INDEX IF NOT EXISTS idx_rounds_guild_id ON rounds(guild_id);
				CREATE INDEX IF NOT EXISTS idx_rounds_created_by ON rounds(created_by);
			`); err != nil {
				return fmt.Errorf("failed to create indices: %w", err)
			}
			return nil
		})
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Rolling back missing indices...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `
				DROP INDEX IF EXISTS idx_round_groups_round_id;
				DROP INDEX IF EXISTS idx_round_group_members_group_id;
				DROP INDEX IF EXISTS idx_rounds_guild_id;
				DROP INDEX IF EXISTS idx_rounds_created_by;
			`); err != nil {
				return fmt.Errorf("failed to drop indices: %w", err)
			}
			return nil
		})
	})
}
