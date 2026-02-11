package leaderboardmigrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Adding tier and opponents columns to leaderboard_point_history...")

		_, err := db.NewRaw("ALTER TABLE leaderboard_point_history ADD COLUMN IF NOT EXISTS tier VARCHAR(10) DEFAULT ''").Exec(ctx)
		if err != nil {
			return err
		}
		_, err = db.NewRaw("ALTER TABLE leaderboard_point_history ADD COLUMN IF NOT EXISTS opponents INTEGER DEFAULT 0").Exec(ctx)
		if err != nil {
			return err
		}

		fmt.Println("Point history audit fields added successfully!")
		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Removing tier and opponents columns from leaderboard_point_history...")

		_, err := db.NewRaw("ALTER TABLE leaderboard_point_history DROP COLUMN IF EXISTS tier").Exec(ctx)
		if err != nil {
			return err
		}
		_, err = db.NewRaw("ALTER TABLE leaderboard_point_history DROP COLUMN IF EXISTS opponents").Exec(ctx)
		if err != nil {
			return err
		}

		fmt.Println("Point history audit fields removed successfully!")
		return nil
	})
}
