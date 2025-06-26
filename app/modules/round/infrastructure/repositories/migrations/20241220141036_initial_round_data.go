package roundmigrations

import (
	"context"
	"fmt"

	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Creating round table...")

		// Create the round table using Bun model
		_, err := db.NewCreateTable().Model((*rounddb.Round)(nil)).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to create round table: %w", err)
		}

		fmt.Println("Round table created successfully!")
		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Rolling back round table...")

		// Drop round table using Bun model
		_, err := db.NewDropTable().Model((*rounddb.Round)(nil)).IfExists().Cascade().Exec(ctx)
		if err != nil {
			return fmt.Errorf("failed to drop round table: %w", err)
		}

		fmt.Println("Round table dropped successfully!")
		return nil
	})
}
