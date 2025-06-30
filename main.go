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

	// Auto-run migrations only if explicitly enabled (for development)
	if os.Getenv("AUTO_MIGRATE") == "true" {
		fmt.Println("DEBUG: AUTO_MIGRATE=true, running migrations on startup...")
		runMigrations()
		fmt.Println("DEBUG: Migrations completed, starting main application...")
	}

	// Create initial context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// --- Configuration Loading ---
	fmt.Println("DEBUG: Starting configuration loading...")
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("DEBUG: Config loaded successfully. NATS_URL: %s, TEMPO_ENDPOINT: %s\n", cfg.NATS.URL, cfg.Observability.TempoEndpoint)

	// --- Observability Initialization ---
	fmt.Println("DEBUG: Starting observability initialization...")
	obsConfig := config.ToObsConfig(cfg)
	obsConfig.Version = "1.0.0"

	obs, err := observability.Init(ctx, obsConfig)
	if err != nil {
		fmt.Printf("DEBUG: Observability initialization failed: %v\n", err)
		fmt.Printf("DEBUG: Config details - Tempo: %s, Loki: %s, Metrics: %s\n",
			obsConfig.TempoEndpoint, obsConfig.LokiURL, obsConfig.MetricsAddress)
		os.Exit(1)
	}
	fmt.Println("DEBUG: Observability initialized successfully")

	logger := obs.Provider.Logger
	fmt.Println("DEBUG: About to initialize application...")

	// --- Application Initialization ---
	application := &app.App{}
	if err := application.Initialize(ctx, cfg, *obs); err != nil {
		fmt.Printf("DEBUG: Application initialization failed: %v\n", err)
		logger.Error("Failed to initialize application", attr.Error(err))
		os.Exit(1)
	}
	fmt.Println("DEBUG: Application initialized successfully")
	logger.Info("Application initialized successfully")

	// --- Graceful Shutdown Setup ---
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	cleanShutdown := make(chan struct{})

	// Goroutine to handle signals and initiate shutdown
	go func() {
		select {
		case sig := <-interrupt:
			fmt.Printf("DEBUG: Received signal: %s\n", sig.String())
			logger.Info("Received signal", attr.String("signal", sig.String()))
		case <-ctx.Done():
			fmt.Println("DEBUG: Context was cancelled from elsewhere")
			logger.Info("Application context cancelled")
		}

		fmt.Println("DEBUG: Initiating graceful shutdown...")
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
	fmt.Println("DEBUG: Starting application run loop...")
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("DEBUG: Application panicked: %v\n", r)
				logger.Error("Application panicked", attr.String("panic", fmt.Sprintf("%v", r)))
				cancel()
			}
		}()

		logger.Info("Starting application run loop")
		if err := application.Run(ctx); err != nil && err != context.Canceled {
			fmt.Printf("DEBUG: Application run failed: %v\n", err)
			logger.Error("Application run failed", attr.Error(err))
			cancel()
		} else if err == context.Canceled {
			fmt.Println("DEBUG: Application run stopped due to context cancellation")
			logger.Info("Application run stopped due to context cancellation")
		} else {
			fmt.Println("DEBUG: Application run loop finished normally (this should not happen unless context cancelled)")
			logger.Info("Application run loop finished")
		}
	}()

	logger.Info("Application is running. Press Ctrl+C to gracefully shut down.")
	fmt.Println("DEBUG: Application startup complete, waiting for shutdown signal...")

	// --- Wait for Shutdown Signal ---
	fmt.Println("DEBUG: Waiting for shutdown signal...")
	<-ctx.Done()
	fmt.Println("DEBUG: Context cancelled, beginning shutdown sequence...")

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
