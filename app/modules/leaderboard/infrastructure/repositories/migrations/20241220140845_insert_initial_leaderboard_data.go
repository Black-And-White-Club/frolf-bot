package leaderboardmigrations

import (
	"context"
	"fmt"

	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Creating leaderboard table...")

		if _, err := db.NewCreateTable().Model((*leaderboarddb.Leaderboard)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}

		// Insert initial leaderboard data using built-in UUID function
		if _, err := db.Exec(`INSERT INTO leaderboards (leaderboard_data, is_active, update_id) VALUES ('[]', true, gen_random_uuid())`); err != nil {
			return err
		}

		fmt.Println("Leaderboard table created successfully!")
		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Dropping leaderboard table...")

		if _, err := db.NewDropTable().Model((*leaderboarddb.Leaderboard)(nil)).IfExists().Exec(ctx); err != nil {
			return err
		}

		fmt.Println("Leaderboard table dropped successfully!")
		return nil
	})
}
