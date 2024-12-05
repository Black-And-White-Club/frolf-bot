package migrations

import (
	"context"
	"fmt"

	"github.com/Black-And-White-Club/tcr-bot/app/models"
	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Creating tables...")

		models := []interface{}{
			(*models.User)(nil),
			(*models.Leaderboard)(nil),
			(*models.Round)(nil),
			(*models.Participant)(nil),
			(*models.RoundScore)(nil),
		}

		for _, model := range models {
			// Add IfNotExists() for initial migration
			if _, err := db.NewCreateTable().Model(model).IfNotExists().Exec(ctx); err != nil {
				return err
			}
		}

		fmt.Println("Tables created successfully!")
		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Dropping tables...")

		// Drop tables in reverse order to avoid foreign key constraints
		models := []interface{}{
			(*models.RoundScore)(nil),
			(*models.Participant)(nil),
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
