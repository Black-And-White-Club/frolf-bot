package migrations

import (
	"context"
	"fmt"

	"github.com/Black-And-White-Club/tcr-bot/models"
	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Creating tables...")

		models := []interface{}{
			(*models.User)(nil),
			(*models.Leaderboard)(nil), // This will create the `leaderboards` table
			(*models.Round)(nil),
			(*models.Score)(nil),
		}

		for _, model := range models {
			if _, err := db.NewCreateTable().Model(model).IfNotExists().Exec(ctx); err != nil {
				return err
			}
		}

		// Call createInitialData after tables are created
		if err := createInitialData(ctx, db); err != nil {
			return fmt.Errorf("failed to create initial data: %w", err)
		}

		fmt.Println("Tables created successfully!")
		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Dropping tables...")

		// Drop tables in reverse order to avoid foreign key constraints
		models := []interface{}{
			(*models.Score)(nil),
			(*models.Round)(nil),
			(*models.Leaderboard)(nil),
			(*models.User)(nil),
		}

		for _, model := range models {
			if _, err := db.NewDropTable().Model(model).IfExists().Exec(ctx); err != nil {
				return err
			}
		}

		fmt.Println("Tables dropped successfully!")
		return nil
	})
}

func createInitialData(ctx context.Context, db *bun.DB) error {
	initialLeaderboard := &models.Leaderboard{
		LeaderboardData: make(map[int]string), // Initialize LeaderboardData as an empty map
		Active:          true,
	}
	if _, err := db.NewInsert().Model(initialLeaderboard).Exec(ctx); err != nil {
		return err
	}
	return nil
}
