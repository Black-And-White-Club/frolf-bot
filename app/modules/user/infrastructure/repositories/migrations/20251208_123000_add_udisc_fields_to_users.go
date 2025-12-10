package usermigrations

import (
	"context"
	"fmt"

	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Adding UDisc fields to users table...")

		// Add udisc_username (normalized on insert/update)
		if _, err := db.NewAddColumn().Model((*userdb.User)(nil)).ColumnExpr("udisc_username TEXT NULL").Exec(ctx); err != nil {
			return err
		}

		// Add udisc_name (normalized on insert/update) - name shown on casual rounds
		if _, err := db.NewAddColumn().Model((*userdb.User)(nil)).ColumnExpr("udisc_name TEXT NULL").Exec(ctx); err != nil {
			return err
		}

		// Create indexes for faster lookups
		if _, err := db.NewCreateIndex().Model((*userdb.User)(nil)).Index("users_udisc_username_idx").Column("udisc_username").IfNotExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewCreateIndex().Model((*userdb.User)(nil)).Index("users_udisc_name_idx").Column("udisc_name").IfNotExists().Exec(ctx); err != nil {
			return err
		}

		// Partial uniqueness on (guild_id, udisc_username) when username is present
		if _, err := db.ExecContext(ctx, "CREATE UNIQUE INDEX IF NOT EXISTS users_guild_username_unique ON users (guild_id, udisc_username) WHERE udisc_username IS NOT NULL"); err != nil {
			return err
		}

		fmt.Println("UDisc fields added successfully!")
		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Dropping UDisc fields from users table...")

		if _, err := db.NewDropIndex().Model((*userdb.User)(nil)).Index("users_udisc_username_idx").IfExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.NewDropIndex().Model((*userdb.User)(nil)).Index("users_udisc_name_idx").IfExists().Exec(ctx); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, "DROP INDEX IF EXISTS users_guild_username_unique"); err != nil {
			return err
		}

		if _, err := db.ExecContext(ctx, "ALTER TABLE users DROP COLUMN IF EXISTS udisc_username"); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, "ALTER TABLE users DROP COLUMN IF EXISTS udisc_name"); err != nil {
			return err
		}

		fmt.Println("UDisc fields dropped successfully!")
		return nil
	})
}
