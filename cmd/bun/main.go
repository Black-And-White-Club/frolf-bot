package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/Black-And-White-Club/frolf-bot/app/shared/migrationrunner"
	"github.com/Black-And-White-Club/frolf-bot/config"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/migrate"
	"github.com/urfave/cli/v2"
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

	migrators := migrationrunner.BuildBunMigrators(db)

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
					return initModules(c.Context, os.Stdout, migrationrunner.AsModuleMigrators(migrators))
				},
			},
			{
				Name:  "migrate",
				Usage: "migrate database",
				Action: func(c *cli.Context) error {
					results, err := migrationrunner.MigrateModules(c.Context, migrationrunner.AsModuleMigrators(migrators))
					if err != nil {
						return err
					}
					for _, result := range results {
						if result.Group == nil || result.Group.IsZero() {
							fmt.Printf("No new migrations to run for module: %s\n", result.Module)
						} else {
							fmt.Printf("Migrated module: %s to %s\n", result.Module, result.Group)
						}
					}
					return nil
				},
			},
			{
				Name:  "rollback",
				Usage: "rollback the last migration group",
				Action: func(c *cli.Context) error {
					results, err := migrationrunner.RollbackModules(c.Context, migrationrunner.AsModuleMigrators(migrators))
					if err != nil {
						return err
					}
					for _, result := range results {
						if result.Group == nil || result.Group.IsZero() {
							fmt.Printf("No groups to roll back for module: %s\n", result.Module)
						} else {
							fmt.Printf("Rolled back module: %s to %s\n", result.Module, result.Group)
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
					moduleNames, err := orderedModuleNames(migrators, false)
					if err != nil {
						return err
					}
					for _, moduleName := range moduleNames {
						migrator := migrators[moduleName]
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
	return migrationrunner.OrderedModuleNames(migrators, reverse)
}

func initModules(ctx context.Context, out io.Writer, migrators map[string]migrationrunner.ModuleMigrator) error {
	moduleNames, err := migrationrunner.OrderedModuleNames(migrators, false)
	if err != nil {
		return err
	}

	for _, moduleName := range moduleNames {
		if _, err := fmt.Fprintf(out, "Initializing migrations for module: %s\n", moduleName); err != nil {
			return err
		}
		if err := migrators[moduleName].Init(ctx); err != nil {
			return fmt.Errorf("init %s migrations: %w", moduleName, err)
		}
	}

	return nil
}
