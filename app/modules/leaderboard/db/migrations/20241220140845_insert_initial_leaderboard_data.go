package leaderboardmigrations

import (
	"context"
	"fmt"

	leaderboarddb "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/db"
	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Creating leaderboard table...")

		if _, err := db.NewCreateTable().Model((*leaderboarddb.Leaderboard)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}

		// Insert initial leaderboard data
		initialLeaderboard := &leaderboarddb.Leaderboard{
			LeaderboardData: make(map[int]string),
			Active:          true,
		}
		if _, err := db.NewInsert().Model(initialLeaderboard).Exec(ctx); err != nil {
			return err
		}

		fmt.Println("Leaderboard table created successfully!")

		// Call createInitialData after tables are created
		if err := createInitialData(ctx, db); err != nil {
			return fmt.Errorf("failed to create initial data: %w", err)
		}
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

func createInitialData(ctx context.Context, db *bun.DB) error {
	initialLeaderboard := &leaderboarddb.Leaderboard{
		LeaderboardData: make(map[int]string), // Initialize LeaderboardData as an empty map
		Active:          true,
	}
	if _, err := db.NewInsert().Model(initialLeaderboard).Exec(ctx); err != nil {
		return err
	}
	return nil
}
