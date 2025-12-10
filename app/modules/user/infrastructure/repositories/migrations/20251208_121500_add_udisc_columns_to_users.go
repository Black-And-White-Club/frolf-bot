package usermigrations

import (
	"context"
	"fmt"

	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Adding UDisc columns to users table...")

		if _, err := db.NewAddColumn().Model((*userdb.User)(nil)).ColumnExpr("udisc_username TEXT NULL").Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewAddColumn().Model((*userdb.User)(nil)).ColumnExpr("udisc_display_name TEXT NULL").Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewAddColumn().Model((*userdb.User)(nil)).ColumnExpr("normalized_username TEXT NOT NULL DEFAULT ''").Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewAddColumn().Model((*userdb.User)(nil)).ColumnExpr("normalized_display_name TEXT NOT NULL DEFAULT ''").Exec(ctx); err != nil {
			return err
		}

		fmt.Println("UDisc columns added successfully!")
		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Dropping UDisc columns from users table...")

		if _, err := db.NewDropColumn().Model((*userdb.User)(nil)).Column("udisc_username").Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewDropColumn().Model((*userdb.User)(nil)).Column("udisc_display_name").Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewDropColumn().Model((*userdb.User)(nil)).Column("normalized_username").Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewDropColumn().Model((*userdb.User)(nil)).Column("normalized_display_name").Exec(ctx); err != nil {
			return err
		}

		fmt.Println("UDisc columns dropped successfully!")
		return nil
	})
}
