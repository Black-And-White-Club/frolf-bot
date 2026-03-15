package testutils

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/uptrace/bun"

	"github.com/Black-And-White-Club/frolf-bot/app/shared/migrationrunner"
)

// runMigrationsWithConnStr runs all module migrations with an optional connection string for River
func runMigrationsWithConnStr(db *bun.DB, pgConnStr string) error {
	ctx := context.Background()

	moduleMigrators := migrationrunner.AsModuleMigrators(migrationrunner.BuildBunMigrators(db))

	// Run River queue migrations first (required for queue system)
	if err := runRiverMigrations(ctx, pgConnStr); err != nil {
		return fmt.Errorf("failed to run River migrations: %w", err)
	}

	if err := migrationrunner.InitModules(ctx, moduleMigrators); err != nil {
		return fmt.Errorf("failed to initialize module migrations: %w", err)
	}
	if _, err := migrationrunner.MigrateModules(ctx, moduleMigrators); err != nil {
		return fmt.Errorf("failed to run module migrations: %w", err)
	}

	log.Println("All migrations ran successfully")
	return nil
}

// runRiverMigrations runs River queue system migrations
func runRiverMigrations(ctx context.Context, pgConnStr string) error {
	// Use the provided connection string, or fallback to a default for tests
	connStr := pgConnStr
	if connStr == "" {
		connStr = "postgres://testuser:testpass@localhost/testdb?sslmode=disable"
	}
	if err := migrationrunner.MigrateRiver(ctx, connStr); err != nil {
		return fmt.Errorf("failed to run River migrations: %w", err)
	}

	log.Println("River queue migrations completed successfully")
	return nil
}

// Known application tables (cached to avoid querying information_schema every time)
var appTables = []string{
	"club_challenge_round_links",
	"club_challenges",
	"club_invites",
	"clubs",
	"guild_memberships",
	"users",
	"scores",
	"rounds",
	"guild_configs",
	"league_members",
	"tag_history",
	"leaderboard_round_outcomes",
	"leaderboard_point_history",
	"leaderboard_season_standings",
	"leaderboard_seasons",
	// betting tables
	"betting_audit_log",
	"betting_bets",
	"betting_market_options",
	"betting_markets",
	"betting_wallet_journal",
	"betting_member_settings",
}

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
	return TruncateTables(
		ctx,
		db,
		"league_members",
		"tag_history",
		"leaderboard_round_outcomes",
		"leaderboard_point_history",
		"leaderboard_season_standings",
		"leaderboard_seasons",
	)
}

// CleanRoundIntegrationTables truncates round-related tables
func CleanRoundIntegrationTables(ctx context.Context, db *bun.DB) error {
	// Order matters due to foreign keys - participants first, then rounds
	return TruncateTables(ctx, db, "rounds")
}

// CleanBettingIntegrationTables truncates all betting-related tables
func CleanBettingIntegrationTables(ctx context.Context, db *bun.DB) error {
	return TruncateTables(ctx, db,
		"betting_audit_log",
		"betting_bets",
		"betting_market_options",
		"betting_markets",
		"betting_wallet_journal",
		"betting_member_settings",
	)
}

// CleanAllIntegrationTables truncates all tables for complete isolation between tests
func CleanAllIntegrationTables(ctx context.Context, db *bun.DB) error {
	// Truncate all tables in the correct order to avoid foreign key issues
	return TruncateTables(
		ctx,
		db,
		"club_challenge_round_links",
		"club_challenges",
		"club_invites",
		"clubs",
		"users",
		"scores",
		"league_members",
		"tag_history",
		"leaderboard_round_outcomes",
		"leaderboard_point_history",
		"leaderboard_season_standings",
		"leaderboard_seasons",
		"rounds",
	)
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
