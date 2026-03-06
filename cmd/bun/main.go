package main

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"

	"github.com/Black-And-White-Club/frolf-bot/config"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/migrate"
	"github.com/urfave/cli/v2"

	// Import for migrator creation
	clubmigrations "github.com/Black-And-White-Club/frolf-bot/app/modules/club/infrastructure/repositories/migrations"
	guildmigrations "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/infrastructure/repositories/migrations"
	leaderboardmigrations "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories/migrations"
	roundmigrations "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories/migrations"
	scoremigrations "github.com/Black-And-White-Club/frolf-bot/app/modules/score/infrastructure/repositories/migrations"
	usermigrations "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories/migrations"
)

var dependencyOrderedModules = []string{
	"guild",
	"user",
	"club",
	"round",
	"score",
	"leaderboard",
}

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
		"user":        migrate.NewMigrator(db, usermigrations.Migrations, migrate.WithTableName("bun_migrations_user")),
		"leaderboard": migrate.NewMigrator(db, leaderboardmigrations.Migrations, migrate.WithTableName("bun_migrations_leaderboard")),
		"score":       migrate.NewMigrator(db, scoremigrations.Migrations, migrate.WithTableName("bun_migrations_score")),
		"round":       migrate.NewMigrator(db, roundmigrations.Migrations, migrate.WithTableName("bun_migrations_round")),
		"guild":       migrate.NewMigrator(db, guildmigrations.Migrations, migrate.WithTableName("bun_migrations_guild")),
		"club":        migrate.NewMigrator(db, clubmigrations.Migrations, migrate.WithTableName("bun_migrations_club")),
	}

	cliApp := &cli.App{
		Name: "bun",
		Commands: []*cli.Command{
			newMultiModuleDBCommand(migrators),
		},
	}

	if err := cliApp.Run(append([]string{os.Args[0]}, flag.Args()...)); err != nil {
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
					moduleNames, err := orderedModuleNames(migrators, false)
					if err != nil {
						return err
					}
					for _, moduleName := range moduleNames {
						migrator := migrators[moduleName]
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
					moduleNames, err := orderedModuleNames(migrators, false)
					if err != nil {
						return err
					}
					for _, moduleName := range moduleNames {
						migrator := migrators[moduleName]
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
					moduleNames, err := orderedModuleNames(migrators, true)
					if err != nil {
						return err
					}
					for _, moduleName := range moduleNames {
						migrator := migrators[moduleName]
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

func orderedModuleNames(migrators map[string]*migrate.Migrator, reverse bool) ([]string, error) {
	if len(migrators) == 0 {
		return nil, errors.New("no migrators configured")
	}

	known := make(map[string]struct{}, len(dependencyOrderedModules))
	for _, moduleName := range dependencyOrderedModules {
		known[moduleName] = struct{}{}
	}

	var missing []string
	for _, moduleName := range dependencyOrderedModules {
		if _, ok := migrators[moduleName]; !ok {
			missing = append(missing, moduleName)
		}
	}

	var unknown []string
	for moduleName := range migrators {
		if _, ok := known[moduleName]; !ok {
			unknown = append(unknown, moduleName)
		}
	}

	slices.Sort(missing)
	slices.Sort(unknown)

	if len(missing) > 0 || len(unknown) > 0 {
		return nil, fmt.Errorf("invalid migrator set: missing=%v unknown=%v", missing, unknown)
	}

	moduleNames := make([]string, len(dependencyOrderedModules))
	copy(moduleNames, dependencyOrderedModules)

	if reverse {
		slices.Reverse(moduleNames)
	}

	return moduleNames, nil
}
