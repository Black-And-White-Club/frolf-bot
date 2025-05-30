package testutils

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"

	leaderboardmigrations "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories/migrations"
	roundmigrations "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories/migrations"
	scoremigrations "github.com/Black-And-White-Club/frolf-bot/app/modules/score/infrastructure/repositories/migrations"
	usermigrations "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories/migrations"
)

// runMigrations runs all module migrations
func runMigrations(db *bun.DB) error {
	ctx := context.Background()

	// Initialize migration tables only once - use any migrations to create the table
	migrator := migrate.NewMigrator(db, usermigrations.Migrations)
	if err := migrator.Init(ctx); err != nil {
		return fmt.Errorf("failed to initialize migration tables: %w", err)
	}

	// Run all module migrations
	for name, m := range map[string]*migrate.Migrations{
		"user":        usermigrations.Migrations,
		"round":       roundmigrations.Migrations,
		"score":       scoremigrations.Migrations,
		"leaderboard": leaderboardmigrations.Migrations,
	} {
		if err := runModuleMigrations(ctx, db, m, name); err != nil {
			return err
		}
	}
	log.Println("All migrations ran successfully")
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
