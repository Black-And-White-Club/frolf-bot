package scoremigrations

import (
	"context"
	"fmt"

	scoredb "github.com/Black-And-White-Club/frolf-bot/app/modules/score/infrastructure/repositories"
	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Creating score table...")

		if _, err := db.NewCreateTable().Model((*scoredb.Score)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}

		fmt.Println("Score table created successfully!")
		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Dropping score table...")

		if _, err := db.NewDropTable().Model((*scoredb.Score)(nil)).IfExists().Exec(ctx); err != nil {
			return err
		}

		fmt.Println("Score table dropped successfully!")
		return nil
	})
}
