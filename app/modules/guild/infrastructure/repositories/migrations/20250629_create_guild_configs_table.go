package migrations

import (
	"context"
	"fmt"

	guilddb "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/infrastructure/repositories"
	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(
		func(ctx context.Context, db *bun.DB) error {
			fmt.Println("Creating guild_configs table...")
			if _, err := db.NewCreateTable().Model((*guilddb.GuildConfig)(nil)).IfNotExists().Exec(ctx); err != nil {
				return err
			}
			fmt.Println("guild_configs table created successfully!")
			return nil
		},
		func(ctx context.Context, db *bun.DB) error {
			fmt.Println("Dropping guild_configs table...")
			if _, err := db.NewDropTable().Model((*guilddb.GuildConfig)(nil)).IfExists().Cascade().Exec(ctx); err != nil {
				return err
			}
			fmt.Println("guild_configs table dropped successfully!")
			return nil
		},
	)
}
