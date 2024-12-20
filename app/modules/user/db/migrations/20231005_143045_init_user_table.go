package usermigrations

import (
	"context"
	"fmt"

	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/db"
	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Creating user table...")

		if _, err := db.NewCreateTable().Model((*userdb.User)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}

		fmt.Println("User table created successfully!")
		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Dropping user table...")

		if _, err := db.NewDropTable().Model((*userdb.User)(nil)).IfExists().Exec(ctx); err != nil {
			return err
		}

		fmt.Println("User table dropped successfully!")
		return nil
	})
}
