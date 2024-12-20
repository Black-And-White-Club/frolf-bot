package roundmigrations

import (
	"context"
	"fmt"

	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Creating round table...")

		if _, err := db.NewCreateTable().Model((*rounddb.Round)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}

		fmt.Println("Round table created successfully!")
		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Dropping round table...")

		if _, err := db.NewDropTable().Model((*rounddb.Round)(nil)).IfExists().Exec(ctx); err != nil {
			return err
		}

		fmt.Println("Round table dropped successfully!")
		return nil
	})
}
