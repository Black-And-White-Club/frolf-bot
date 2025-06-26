package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot/app"
	"github.com/Black-And-White-Club/frolf-bot/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/migrate"

	// Import for migrator creation
	leaderboardmigrations "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories/migrations"
	roundmigrations "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories/migrations"
	scoremigrations "github.com/Black-And-White-Club/frolf-bot/app/modules/score/infrastructure/repositories/migrations"
	usermigrations "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories/migrations"
)

func main() {
	// Check for migrate command
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		runMigrations()
		return
	}

	// Create initial context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// --- Configuration Loading ---
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// --- Observability Initialization ---
	obsConfig := config.ToObsConfig(cfg)
	obsConfig.Version = "1.0.0"

	obs, err := observability.Init(ctx, obsConfig)
	if err != nil {
		fmt.Printf("Failed to initialize observability: %v\n", err)
		os.Exit(1)
	}

	logger := obs.Provider.Logger
	logger.Info("Observability initialized successfully")

	// --- Application Initialization ---
	application := &app.App{}
	if err := application.Initialize(ctx, cfg, *obs); err != nil {
		logger.Error("Failed to initialize application", attr.Error(err))
		os.Exit(1)
	}
	logger.Info("Application initialized successfully")

	// --- Graceful Shutdown Setup ---
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	cleanShutdown := make(chan struct{})

	// Goroutine to handle signals and initiate shutdown
	go func() {
		select {
		case sig := <-interrupt:
			logger.Info("Received signal", attr.String("signal", sig.String()))
		case <-ctx.Done():
			logger.Info("Application context cancelled")
		}

		logger.Info("Initiating graceful shutdown...")
		cancel()

		// Create a timeout for the entire shutdown process
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer shutdownCancel()

		// Shutdown application first
		go func() {
			defer close(cleanShutdown)
			logger.Info("Closing application...")
			if err := application.Close(); err != nil {
				logger.Error("Error during application shutdown", attr.Error(err))
			} else {
				logger.Info("Application closed successfully")
			}

			// Shutdown observability components
			logger.Info("Shutting down observability components...")
			obsShutdownCtx, obsShutdownCancel := context.WithTimeout(shutdownCtx, 10*time.Second)
			defer obsShutdownCancel()
			if err := obs.Provider.Shutdown(obsShutdownCtx); err != nil {
				logger.Error("Error shutting down observability", attr.Error(err))
			} else {
				logger.Info("Observability shutdown successful")
			}
		}()

		// Wait for cleanup or timeout
		select {
		case <-shutdownCtx.Done():
			if shutdownCtx.Err() == context.DeadlineExceeded {
				logger.Error("Graceful shutdown timeout reached, forcing exit")
				os.Exit(1)
			}
		case <-cleanShutdown:
			logger.Info("Graceful shutdown completed successfully")
		}
	}()

	// --- Run Application ---
	go func() {
		logger.Info("Starting application run loop")
		if err := application.Run(ctx); err != nil && err != context.Canceled {
			logger.Error("Application run failed", attr.Error(err))
			cancel()
		} else {
			logger.Info("Application run loop finished")
		}
	}()

	logger.Info("Application is running. Press Ctrl+C to gracefully shut down.")

	// --- Wait for Shutdown Signal ---
	<-ctx.Done()

	// --- Final Wait for Cleanup ---
	select {
	case <-cleanShutdown:
		// Shutdown completed cleanly
	case <-time.After(5 * time.Second):
		logger.Warn("Did not receive clean shutdown signal within final wait period.")
	}

	logger.Info("Exiting main.")
	os.Exit(0)
}

// runMigrations handles database migration execution
func runMigrations() {
	fmt.Println("Running database migrations...")

	// Load configuration
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// First, run River migrations
	fmt.Println("Running River queue migrations...")
	dbPool, err := pgxpool.New(context.Background(), cfg.Postgres.DSN)
	if err != nil {
		fmt.Printf("Failed to connect to database for River migrations: %v\n", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	migrator, err := rivermigrate.New(riverpgxv5.New(dbPool), nil)
	if err != nil {
		fmt.Printf("Failed to create River migrator: %v\n", err)
		os.Exit(1)
	}
	_, err = migrator.Migrate(context.Background(), rivermigrate.DirectionUp, &rivermigrate.MigrateOpts{})
	if err != nil {
		fmt.Printf("Failed to run River migrations: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("River migrations completed successfully.")

	// Then run application migrations
	fmt.Println("Running application migrations...")
	pgdb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(cfg.Postgres.DSN)))
	db := bun.NewDB(pgdb, pgdialect.New())
	defer db.Close()

	// Create migrators for all modules
	migrators := map[string]*migrate.Migrator{
		"user":        migrate.NewMigrator(db, usermigrations.Migrations),
		"leaderboard": migrate.NewMigrator(db, leaderboardmigrations.Migrations),
		"score":       migrate.NewMigrator(db, scoremigrations.Migrations),
		"round":       migrate.NewMigrator(db, roundmigrations.Migrations),
	}

	// Initialize and run migrations for each module
	for moduleName, migrator := range migrators {
		fmt.Printf("Initializing migrations for %s module...\n", moduleName)
		if err := migrator.Init(context.Background()); err != nil {
			fmt.Printf("Failed to initialize %s migrations: %v\n", moduleName, err)
			os.Exit(1)
		}

		fmt.Printf("Running migrations for %s module...\n", moduleName)
		group, err := migrator.Migrate(context.Background())
		if err != nil {
			fmt.Printf("Failed to run %s migrations: %v\n", moduleName, err)
			os.Exit(1)
		}

		if group.IsZero() {
			fmt.Printf("No new migrations for %s module\n", moduleName)
		} else {
			fmt.Printf("Successfully migrated %s module to %s\n", moduleName, group)
		}
	}

	fmt.Println("All migrations completed successfully!")
}
