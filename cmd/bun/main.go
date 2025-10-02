package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Black-And-White-Club/frolf-bot/config"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/migrate"
	"github.com/urfave/cli/v2"

	// Import for migrator creation
	guildmigrations "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/infrastructure/repositories/migrations"
	leaderboardmigrations "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories/migrations"
	roundmigrations "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories/migrations"
	scoremigrations "github.com/Black-And-White-Club/frolf-bot/app/modules/score/infrastructure/repositories/migrations"
	usermigrations "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories/migrations"
)

func main() {
	// Load configuration for database connection ONLY
	configFile := flag.String("config", "config.yaml", "Path to the configuration file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	fmt.Printf("Loaded Config: %+v\n", cfg)

	// Database connection using pgdriver
	pgdb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(cfg.Postgres.DSN)))
	db := bun.NewDB(pgdb, pgdialect.New())
	defer db.Close()

	// Create migrators
	migrators := map[string]*migrate.Migrator{
		"user":        migrate.NewMigrator(db, usermigrations.Migrations),
		"leaderboard": migrate.NewMigrator(db, leaderboardmigrations.Migrations),
		"score":       migrate.NewMigrator(db, scoremigrations.Migrations),
		"round":       migrate.NewMigrator(db, roundmigrations.Migrations),
		"guild":       migrate.NewMigrator(db, guildmigrations.Migrations),
	}

	cliApp := &cli.App{
		Name: "bun",
		Commands: []*cli.Command{
			newMultiModuleDBCommand(migrators),
		},
	}

	if err := cliApp.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func newMultiModuleDBCommand(migrators map[string]*migrate.Migrator) *cli.Command {
	return &cli.Command{
		Name:  "migrate",
		Usage: "database migrations",
		Subcommands: []*cli.Command{
			{
				Name:  "init",
				Usage: "create migration tables",
				Action: func(c *cli.Context) error {
					for moduleName, migrator := range migrators {
						fmt.Printf("Initializing migrations for module: %s\n", moduleName)
						if err := migrator.Init(c.Context); err != nil {
							fmt.Printf("Error initializing migrations for module %s: %v\n", moduleName, err)
							return err
						}
					}
					return nil
				},
			},
			{
				Name:  "migrate",
				Usage: "migrate database",
				Action: func(c *cli.Context) error {
					for moduleName, migrator := range migrators {
						fmt.Printf("Running migrations for module: %s\n", moduleName)
						group, err := migrator.Migrate(c.Context)
						if err != nil {
							return err
						}
						if group.IsZero() {
							fmt.Printf("No new migrations to run for module: %s\n", moduleName)
						} else {
							fmt.Printf("Migrated module: %s to %s\n", moduleName, group)
						}
					}
					return nil
				},
			},
			{
				Name:  "rollback",
				Usage: "rollback the last migration group",
				Action: func(c *cli.Context) error {
					for moduleName, migrator := range migrators {
						fmt.Printf("Rolling back migrations for module: %s\n", moduleName)
						group, err := migrator.Rollback(c.Context)
						if err != nil {
							return err
						}
						if group.IsZero() {
							fmt.Printf("No groups to roll back for module: %s\n", moduleName)
						} else {
							fmt.Printf("Rolled back module: %s to %s\n", moduleName, group)
						}
					}
					return nil
				},
			},
			{
				Name:  "create_go",
				Usage: "create Go migration",
				Action: func(c *cli.Context) error {
					moduleName := c.Args().First() // Get module name from args
					migrator, ok := migrators[moduleName]
					if !ok {
						return fmt.Errorf("invalid module name: %s", moduleName)
					}

					name := strings.Join(c.Args().Tail(), "_")
					mf, err := migrator.CreateGoMigration(c.Context, name)
					if err != nil {
						return err
					}
					fmt.Printf("Created migration for module %s: %s (%s)\n", moduleName, mf.Name, mf.Path)
					return nil
				},
			},
			{
				Name:  "create_sql",
				Usage: "create up and down SQL migrations",
				Action: func(c *cli.Context) error {
					moduleName := c.Args().First() // Get module name from args
					migrator, ok := migrators[moduleName]
					if !ok {
						return fmt.Errorf("invalid module name: %s", moduleName)
					}

					name := strings.Join(c.Args().Tail(), "_")
					files, err := migrator.CreateSQLMigrations(c.Context, name)
					if err != nil {
						return err
					}

					for _, mf := range files {
						fmt.Printf("Created migration for module %s: %s (%s)\n", moduleName, mf.Name, mf.Path)
					}

					return nil
				},
			},
			{
				Name:  "status",
				Usage: "print migrations status",
				Action: func(c *cli.Context) error {
					for moduleName, migrator := range migrators {
						ms, err := migrator.MigrationsWithStatus(c.Context)
						if err != nil {
							return err
						}
						fmt.Printf("Migrations for module: %s\n", moduleName)
						fmt.Printf("  %s\n", ms)
						fmt.Printf("  Applied: %s\n", ms.Applied())
						fmt.Printf("  Unapplied: %s\n", ms.Unapplied())
					}
					return nil
				},
			},
		},
	}
}
