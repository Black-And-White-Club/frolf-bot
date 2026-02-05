package testutils

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"

	clubmigrations "github.com/Black-And-White-Club/frolf-bot/app/modules/club/infrastructure/repositories/migrations"
	guildmigrations "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/infrastructure/repositories/migrations"
	leaderboardmigrations "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories/migrations"
	roundmigrations "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories/migrations"
	scoremigrations "github.com/Black-And-White-Club/frolf-bot/app/modules/score/infrastructure/repositories/migrations"
	usermigrations "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories/migrations"
)

// runMigrationsWithConnStr runs all module migrations with an optional connection string for River
func runMigrationsWithConnStr(db *bun.DB, pgConnStr string) error {
	ctx := context.Background()

	// Initialize migration tables only once - use any migrations to create the table
	migrator := migrate.NewMigrator(db, usermigrations.Migrations)
	if err := migrator.Init(ctx); err != nil {
		return fmt.Errorf("failed to initialize migration tables: %w", err)
	}

	// Run River queue migrations first (required for queue system)
	if err := runRiverMigrations(ctx, db, pgConnStr); err != nil {
		return fmt.Errorf("failed to run River migrations: %w", err)
	}

	// Run all module migrations in a deterministic order
	// Order matters due to foreign key constraints (e.g. user -> guild)
	orderedModules := []struct {
		name       string
		migrations *migrate.Migrations
	}{
		{"guild", guildmigrations.Migrations},
		{"user", usermigrations.Migrations},
		{"club", clubmigrations.Migrations},
		{"round", roundmigrations.Migrations},
		{"leaderboard", leaderboardmigrations.Migrations},
		{"score", scoremigrations.Migrations},
	}

	for _, mod := range orderedModules {
		if err := runModuleMigrations(ctx, db, mod.migrations, mod.name); err != nil {
			return err
		}
	}
	log.Println("All migrations ran successfully")
	return nil
}

// runRiverMigrations runs River queue system migrations
func runRiverMigrations(ctx context.Context, db *bun.DB, pgConnStr string) error {
	// Use the provided connection string, or fallback to a default for tests
	connStr := pgConnStr
	if connStr == "" {
		connStr = "postgres://testuser:testpass@localhost/testdb?sslmode=disable"
	}

	// Create pgx pool for River migrations
	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return fmt.Errorf("failed to parse DSN for River migrations: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to create pgx pool for River migrations: %w", err)
	}
	defer pool.Close()

	// Run River migrations
	migrator, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		return fmt.Errorf("failed to create River migrator: %w", err)
	}

	_, err = migrator.Migrate(ctx, rivermigrate.DirectionUp, &rivermigrate.MigrateOpts{})
	if err != nil {
		return fmt.Errorf("failed to run River migrations: %w", err)
	}

	log.Println("River queue migrations completed successfully")
	return nil
}

// runModuleMigrations runs migrations for a specific module
func runModuleMigrations(ctx context.Context, db *bun.DB, migrations *migrate.Migrations, name string) error {
	migrator := migrate.NewMigrator(db, migrations)
	group, err := migrator.Migrate(ctx)
	if err != nil {
		return fmt.Errorf("failed to run %s migrations: %w", name, err)
	}
	if group.ID == 0 {
		log.Printf("No %s migrations to run", name)
	} else {
		log.Printf("Ran %s migrations group #%d", name, group.ID)
	}
	return nil
}

// Known application tables (cached to avoid querying information_schema every time)
var appTables = []string{"guild_memberships", "users", "scores", "leaderboards", "rounds", "guild_configs"}

// CleanupRiverJobs deletes all jobs from the River queue
func CleanupRiverJobs(ctx context.Context, db *bun.DB) error {
	_, err := db.ExecContext(ctx, "DELETE FROM river_job")
	return err
}

// CleanupDatabase truncates all tables in the database to ensure a clean state
func CleanupDatabase(ctx context.Context, db *bun.DB) error {
	if len(appTables) == 0 {
		return nil
	}

	// Truncate all application tables (skip migrations tables)
	// Using CASCADE to handle foreign key constraints automatically
	query := fmt.Sprintf("TRUNCATE TABLE %s CASCADE", strings.Join(appTables, ", "))
	if _, err := db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to truncate tables: %w", err)
	}

	// Clean up River queue jobs
	if err := CleanupRiverJobs(ctx, db); err != nil {
		// Don't fail if table doesn't exist yet
		if !strings.Contains(err.Error(), "does not exist") {
			return fmt.Errorf("failed to cleanup river jobs: %w", err)
		}
	}

	return nil
}

// TruncateTables truncates the specified tables
func TruncateTables(ctx context.Context, db *bun.DB, tables ...string) error {
	if len(tables) == 0 {
		return nil
	}

	var sb strings.Builder
	sb.WriteString("TRUNCATE TABLE ")
	for i, table := range tables {
		sb.WriteString(fmt.Sprintf(`"%s"`, table))
		if i < len(tables)-1 {
			sb.WriteString(", ")
		}
	}
	sb.WriteString(" CASCADE")

	log.Printf("Truncating tables: %s", strings.Join(tables, ", "))
	if _, err := db.ExecContext(ctx, sb.String()); err != nil {
		return fmt.Errorf("failed to truncate tables %v: %w", tables, err)
	}
	return nil
}

// CleanUserIntegrationTables truncates user-related tables
func CleanUserIntegrationTables(ctx context.Context, db *bun.DB) error {
	return TruncateTables(ctx, db, "users")
}

// CleanScoreIntegrationTables truncates score-related tables
func CleanScoreIntegrationTables(ctx context.Context, db *bun.DB) error {
	return TruncateTables(ctx, db, "scores")
}

// CleanLeaderboardIntegrationTables truncates leaderboard-related tables
func CleanLeaderboardIntegrationTables(ctx context.Context, db *bun.DB) error {
	return TruncateTables(ctx, db, "leaderboards")
}

// CleanRoundIntegrationTables truncates round-related tables
func CleanRoundIntegrationTables(ctx context.Context, db *bun.DB) error {
	// Order matters due to foreign keys - participants first, then rounds
	return TruncateTables(ctx, db, "rounds")
}

// CleanAllIntegrationTables truncates all tables for complete isolation between tests
func CleanAllIntegrationTables(ctx context.Context, db *bun.DB) error {
	// Truncate all tables in the correct order to avoid foreign key issues
	return TruncateTables(ctx, db, "users", "scores", "leaderboards", "rounds")
}

// ForceCleanAllTables performs aggressive cleanup including sequences and constraints
func ForceCleanAllTables(ctx context.Context, db *bun.DB) error {
	log.Println("Performing aggressive table cleanup for test isolation")

	// First try normal truncation
	if err := CleanAllIntegrationTables(ctx, db); err != nil {
		return fmt.Errorf("failed to clean tables normally: %w", err)
	}

	// Reset sequences to ensure consistent IDs across tests
	sequences := []string{
		"users_id_seq",
		"scores_id_seq",
		"leaderboards_id_seq",
		"rounds_id_seq",
	}

	for _, seq := range sequences {
		if _, err := db.ExecContext(ctx, fmt.Sprintf("ALTER SEQUENCE %s RESTART WITH 1", seq)); err != nil {
			// Don't fail if sequence doesn't exist
			log.Printf("Warning: Could not reset sequence %s: %v", seq, err)
		}
	}

	return nil
}
