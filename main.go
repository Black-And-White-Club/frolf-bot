package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	pprof "net/http/pprof"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot/app"
	"github.com/Black-And-White-Club/frolf-bot/app/shared/migrationrunner"
	"github.com/Black-And-White-Club/frolf-bot/config"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

// --- Optional pprof server for on-demand profiling ---
func startPprofIfEnabled() {
	if os.Getenv("PPROF_ENABLED") != "true" {
		return
	}
	addr := os.Getenv("PPROF_ADDR")
	if addr == "" {
		addr = "127.0.0.1:6060"
	} else if strings.HasPrefix(addr, ":") {
		addr = "127.0.0.1" + addr
	}

	if !isLoopbackAddr(addr) {
		log.Printf("pprof warning: non-loopback bind address %s may expose profiling data", addr)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	var handler http.Handler = mux
	if token := strings.TrimSpace(os.Getenv("PPROF_AUTH_TOKEN")); token != "" {
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer "+token {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			mux.ServeHTTP(w, r)
		})
	}

	go func() {
		log.Printf("pprof enabled on %s", addr)
		if err := http.ListenAndServe(addr, handler); err != nil {
			log.Printf("pprof server stopped: %v", err)
		}
	}()
}

func isLoopbackAddr(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// validDBName matches safe PostgreSQL database names: start with letter or underscore,
// followed by letters, digits, or underscores only.
var validDBName = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// Version is set at build time via ldflags in CI.
var Version = "dev"

func runtimeServiceVersion(configured string) string {
	if value := strings.TrimSpace(os.Getenv("SERVICE_VERSION")); value != "" {
		return value
	}
	if value := strings.TrimSpace(configured); value != "" {
		return value
	}
	if value := strings.TrimSpace(Version); value != "" {
		return value
	}
	return "dev"
}

func main() {
	// Optionally start pprof for profiling
	startPprofIfEnabled()

	// Check for migrate command
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		runMigrations()
		return
	}

	// Auto-run migrations only if explicitly enabled (for development)
	if os.Getenv("AUTO_MIGRATE") == "true" {
		runMigrations()
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
	obsConfig.Version = runtimeServiceVersion(obsConfig.Version)

	obs, err := observability.Init(ctx, obsConfig)
	if err != nil {
		fmt.Printf("Failed to initialize observability: %v\n", err)
		os.Exit(1)
	}

	logger := obs.Provider.Logger

	// --- Application Initialization ---
	application := &app.App{}
	if err := application.Initialize(ctx, cfg, *obs); err != nil {
		logger.Error("Failed to initialize application", attr.Error(err))
		os.Exit(1)
	}
	logger.Info("Application initialized successfully")

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer signal.Stop(interrupt)

	// --- Run Application ---
	runErrCh := make(chan error, 1)
	reportRunError := func(err error) {
		select {
		case runErrCh <- err:
		default:
		}
	}
	go func() {
		defer close(runErrCh)
		defer func() {
			if r := recover(); r != nil {
				reportRunError(fmt.Errorf("application panicked: %v", r))
			}
		}()

		logger.Info("Starting application run loop")
		if err := application.Run(ctx); err != nil && err != context.Canceled {
			reportRunError(fmt.Errorf("application run failed: %w", err))
		} else if err == context.Canceled {
			logger.Info("Application run stopped due to context cancellation")
		} else {
			logger.Info("Application run loop finished")
		}
	}()

	logger.Info("Application is running. Press Ctrl+C to gracefully shut down.")

	exitCode := 0
	shutdownReason := "context canceled"

	select {
	case sig := <-interrupt:
		shutdownReason = fmt.Sprintf("signal: %s", sig.String())
		logger.Info("Received signal", attr.String("signal", sig.String()))
		cancel()
	case err, ok := <-runErrCh:
		shutdownReason = "application run loop exited"
		cancel()
		if ok && err != nil {
			logger.Error("Application run loop failed", attr.Error(err))
			exitCode = 1
		}
	case <-ctx.Done():
		shutdownReason = "application context canceled"
	}

	logger.Info("Initiating graceful shutdown", attr.String("reason", shutdownReason))

	// Wait for run loop to stop after cancellation.
	select {
	case err, ok := <-runErrCh:
		if ok && err != nil {
			logger.Error("Application run loop failed during shutdown", attr.Error(err))
			exitCode = 1
		}
	case <-time.After(5 * time.Second):
		logger.Warn("Timed out waiting for application run loop to stop")
	}

	// Shutdown application resources from main only.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer shutdownCancel()

	closeDone := make(chan error, 1)
	go func() {
		closeDone <- application.Close()
	}()

	select {
	case err := <-closeDone:
		if err != nil {
			logger.Error("Error during application shutdown", attr.Error(err))
			exitCode = 1
		} else {
			logger.Info("Application closed successfully")
		}
	case <-shutdownCtx.Done():
		logger.Error("Graceful shutdown timeout reached", attr.Error(shutdownCtx.Err()))
		exitCode = 1
	}

	// Shutdown observability components after application resources.
	logger.Info("Shutting down observability components...")
	obsShutdownCtx, obsShutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer obsShutdownCancel()
	if err := obs.Provider.Shutdown(obsShutdownCtx); err != nil {
		logger.Error("Error shutting down observability", attr.Error(err))
		exitCode = 1
	} else {
		logger.Info("Observability shutdown successful")
	}

	if exitCode != 0 {
		os.Exit(exitCode)
	}
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

	// Ensure database exists
	ensureDatabaseExists(cfg.Postgres.DSN)

	// First, run River migrations
	fmt.Println("Running River queue migrations...")
	if err := migrationrunner.MigrateRiver(context.Background(), cfg.Postgres.DSN); err != nil {
		fmt.Printf("Failed to run River migrations: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("River migrations completed successfully.")

	// Then run application migrations
	fmt.Println("Running application migrations...")
	pgdb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(cfg.Postgres.DSN)))
	db := bun.NewDB(pgdb, pgdialect.New())
	defer db.Close()

	bunMigrators := migrationrunner.BuildSharedTableMigrators(db)
	moduleMigrators := migrationrunner.AsModuleMigrators(bunMigrators)

	if err := migrationrunner.InitModules(context.Background(), moduleMigrators); err != nil {
		fmt.Printf("Failed to initialize application migrations: %v\n", err)
		os.Exit(1)
	}

	results, err := migrationrunner.MigrateModules(context.Background(), moduleMigrators)
	if err != nil {
		fmt.Printf("Failed to run application migrations: %v\n", err)
		os.Exit(1)
	}
	for _, result := range results {
		if result.Group == nil || result.Group.IsZero() {
			fmt.Printf("No new migrations for %s module\n", result.Module)
			continue
		}
		fmt.Printf("Successfully migrated %s module to %s\n", result.Module, result.Group)
	}

	fmt.Println("All migrations completed successfully!")
}

func ensureDatabaseExists(dsn string) {
	// Parse DSN to get database name and connection info
	// Assuming DSN format: postgres://user:pass@host:port/dbname?options
	// We need to connect to "postgres" database to create the new database

	// Simple parsing (can be improved with a library if needed)
	parts := strings.Split(dsn, "/")
	if len(parts) < 4 {
		fmt.Println("Invalid DSN format, skipping database creation check")
		return
	}

	dbNameWithArgs := parts[len(parts)-1]
	dbName := strings.Split(dbNameWithArgs, "?")[0]

	if !validDBName.MatchString(dbName) {
		fmt.Printf("ensureDatabaseExists: unsafe database name %q, skipping\n", dbName)
		return
	}

	// Replace dbname with "postgres" for the initial connection
	postgresDSN := strings.Replace(dsn, "/"+dbName, "/postgres", 1)

	fmt.Printf("Checking if database '%s' exists...\n", dbName)

	// Use pgdriver.NewConnector to create a connector, then sql.OpenDB
	connector := pgdriver.NewConnector(pgdriver.WithDSN(postgresDSN))
	db := sql.OpenDB(connector)

	if err := db.Ping(); err != nil {
		fmt.Printf("Failed to connect to postgres database: %v\n", err)
		return
	}
	defer db.Close()

	// Check if database exists
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", dbName).Scan(&exists)
	if err != nil {
		fmt.Printf("Failed to check if database exists: %v\n", err)
		return
	}

	if !exists {
		fmt.Printf("Database '%s' does not exist. Creating...\n", dbName)
		_, err = db.Exec(fmt.Sprintf("CREATE DATABASE \"%s\"", dbName))
		if err != nil {
			fmt.Printf("Failed to create database: %v\n", err)
			// Don't exit, let the migration fail naturally if this didn't work
			return
		}
		fmt.Printf("Database '%s' created successfully.\n", dbName)
	} else {
		fmt.Printf("Database '%s' already exists.\n", dbName)
	}
}
