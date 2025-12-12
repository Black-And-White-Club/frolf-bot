package migrations

import (
	"context"
	"fmt"

	guilddb "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/infrastructure/repositories"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

var Migrations = migrate.NewMigrations()

// Migration for creating the guild_configs table using Bun model
func CreateGuildConfigsTable(ctx context.Context, db *bun.DB) error {
	fmt.Println("Creating guild_configs table...")
	_, err := db.NewCreateTable().Model((*guilddb.GuildConfig)(nil)).IfNotExists().Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to create guild_configs table: %w", err)
	}
	fmt.Println("guild_configs table created successfully!")
	return nil
}

// Migration for dropping the guild_configs table
func DropGuildConfigsTable(ctx context.Context, db *bun.DB) error {
	fmt.Println("Dropping guild_configs table...")
	_, err := db.NewDropTable().Model((*guilddb.GuildConfig)(nil)).IfExists().Cascade().Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to drop guild_configs table: %w", err)
	}
	fmt.Println("guild_configs table dropped successfully!")
	return nil
}

func init() {
	if err := Migrations.DiscoverCaller(); err != nil {
		panic(err)
	}
}
