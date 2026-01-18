package usermigrations

import (
	"context"
	"fmt"
	"os"

	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/uptrace/bun"
)

func init() {
	if err := Migrations.DiscoverCaller(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: migration caller discovery failed: %v\n", err)
		// proceed without DiscoverCaller
	}
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Adding UDisc fields to users table...")

		// NOTE: The users table may already contain these columns (fresh DBs are created
		// from the current bun model). Keep this migration safe to run regardless.
		if _, err := db.ExecContext(ctx, "ALTER TABLE users ADD COLUMN IF NOT EXISTS udisc_username TEXT NULL"); err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, "ALTER TABLE users ADD COLUMN IF NOT EXISTS udisc_name TEXT NULL"); err != nil {
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
