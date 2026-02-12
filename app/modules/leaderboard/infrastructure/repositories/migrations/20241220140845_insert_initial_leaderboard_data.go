package leaderboardmigrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Creating leaderboard table...")

		_, err := db.NewRaw(`
			CREATE TABLE IF NOT EXISTS leaderboards (
				id              bigserial    PRIMARY KEY,
				leaderboard_data jsonb       NOT NULL,
				is_active       boolean      NOT NULL DEFAULT true,
				update_source   text         NULL,
				update_id       uuid         NULL,
				guild_id        text         NOT NULL
			)
		`).Exec(ctx)
		if err != nil {
			return err
		}

		fmt.Println("Leaderboard table created successfully!")
		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Dropping leaderboard table...")

		if _, err := db.NewRaw("DROP TABLE IF EXISTS leaderboards").Exec(ctx); err != nil {
			return err
		}

		fmt.Println("Leaderboard table dropped successfully!")
		return nil
	})
}
