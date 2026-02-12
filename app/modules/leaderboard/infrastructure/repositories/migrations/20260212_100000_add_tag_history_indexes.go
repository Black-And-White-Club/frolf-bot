package leaderboardmigrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Adding tag history query indexes...")

		// Index for "show me the full history of tag N across all holders"
		_, err := db.NewRaw(`
			CREATE INDEX IF NOT EXISTS idx_tag_history_guild_tag_timeline
			ON tag_history (guild_id, tag_number, created_at ASC)
		`).Exec(ctx)
		if err != nil {
			return fmt.Errorf("create idx_tag_history_guild_tag_timeline: %w", err)
		}

		// Index for "show me all tag changes where member received a tag"
		_, err = db.NewRaw(`
			CREATE INDEX IF NOT EXISTS idx_tag_history_new_member_timeline
			ON tag_history (guild_id, new_member_id, created_at ASC)
		`).Exec(ctx)
		if err != nil {
			return fmt.Errorf("create idx_tag_history_new_member_timeline: %w", err)
		}

		// Index for "show me all tag changes where member lost a tag"
		_, err = db.NewRaw(`
			CREATE INDEX IF NOT EXISTS idx_tag_history_old_member_timeline
			ON tag_history (guild_id, old_member_id, created_at ASC)
			WHERE old_member_id IS NOT NULL
		`).Exec(ctx)
		if err != nil {
			return fmt.Errorf("create idx_tag_history_old_member_timeline: %w", err)
		}

		// Composite index for graph range queries (chart generation)
		_, err = db.NewRaw(`
			CREATE INDEX IF NOT EXISTS idx_tag_history_guild_created
			ON tag_history (guild_id, created_at ASC)
		`).Exec(ctx)
		if err != nil {
			return fmt.Errorf("create idx_tag_history_guild_created: %w", err)
		}

		fmt.Println("Tag history query indexes created successfully!")
		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Dropping tag history query indexes...")

		_, _ = db.NewRaw("DROP INDEX IF EXISTS idx_tag_history_guild_tag_timeline").Exec(ctx)
		_, _ = db.NewRaw("DROP INDEX IF EXISTS idx_tag_history_new_member_timeline").Exec(ctx)
		_, _ = db.NewRaw("DROP INDEX IF EXISTS idx_tag_history_old_member_timeline").Exec(ctx)
		_, _ = db.NewRaw("DROP INDEX IF EXISTS idx_tag_history_guild_created").Exec(ctx)

		fmt.Println("Tag history query indexes dropped successfully!")
		return nil
	})
}
