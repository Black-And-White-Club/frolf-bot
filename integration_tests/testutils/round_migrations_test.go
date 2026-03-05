package testutils

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"

	roundmigrations "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories/migrations"
	"github.com/Black-And-White-Club/frolf-bot/db/bundb"
)

func setupIsolatedPostgresDB(t *testing.T) (*bun.DB, context.Context) {
	t.Helper()

	configureLocalDockerAutodetect()

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	t.Cleanup(cancel)

	_, _, connStr, _, err := globalPool.Acquire(ctx)
	if err != nil {
		t.Fatalf("failed to acquire shared postgres container: %v", err)
	}
	t.Cleanup(globalPool.Release)

	adminDB, err := sql.Open("pgx", connStr)
	if err != nil {
		t.Fatalf("failed to open admin sql db: %v", err)
	}
	t.Cleanup(func() {
		_ = adminDB.Close()
	})

	dbName := sanitizeDBName(fmt.Sprintf("round_migration_%d", time.Now().UnixNano()))
	if _, err := adminDB.ExecContext(ctx, fmt.Sprintf(`CREATE DATABASE "%s"`, dbName)); err != nil {
		t.Fatalf("failed creating isolated database %q: %v", dbName, err)
	}

	testConnStr, err := withDatabaseName(connStr, dbName)
	if err != nil {
		t.Fatalf("failed building isolated db connection string: %v", err)
	}

	sqlDB, err := sql.Open("pgx", testConnStr)
	if err != nil {
		t.Fatalf("failed to open isolated sql db: %v", err)
	}
	t.Cleanup(func() {
		_ = sqlDB.Close()
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		_, _ = adminDB.ExecContext(
			cleanupCtx,
			`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid()`,
			dbName,
		)
		_, _ = adminDB.ExecContext(cleanupCtx, fmt.Sprintf(`DROP DATABASE IF EXISTS "%s"`, dbName))
	})

	db := bundb.BunDB(sqlDB)
	t.Cleanup(func() {
		_ = db.Close()
	})

	return db, context.Background()
}

func sanitizeDBName(name string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9_]+`)
	cleaned := re.ReplaceAllString(name, "_")
	if cleaned == "" {
		return "round_migration_tmp"
	}
	return cleaned
}

func withDatabaseName(connStr, dbName string) (string, error) {
	parsed, err := url.Parse(connStr)
	if err != nil {
		return "", err
	}
	parsed.Path = "/" + dbName
	return parsed.String(), nil
}

func runSingleRoundMigration(t *testing.T, ctx context.Context, db *bun.DB, migrationNameContains string) {
	t.Helper()

	migrator := migrate.NewMigrator(db, roundmigrations.Migrations, migrate.WithUpsert(true))
	if err := migrator.Init(ctx); err != nil {
		t.Fatalf("failed to init round migrator: %v", err)
	}

	migrations, err := migrator.MigrationsWithStatus(ctx)
	if err != nil {
		t.Fatalf("failed to list round migrations: %v", err)
	}

	var targetName string
	for _, migration := range migrations {
		if strings.Contains(migration.Name, migrationNameContains) || strings.Contains(migration.Comment, migrationNameContains) {
			targetName = migration.Name
			break
		}
	}
	if targetName == "" {
		t.Fatalf("migration containing %q not found", migrationNameContains)
	}

	if err := migrator.RunMigration(ctx, targetName); err != nil {
		t.Fatalf("failed to run migration %q: %v", targetName, err)
	}
}

func TestRoundMigration_AddRoundGroups_BackfillsLegacyRows(t *testing.T) {
	db, ctx := setupIsolatedPostgresDB(t)

	_, err := db.ExecContext(ctx, `
		CREATE TABLE rounds (
			id UUID PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT NULL,
			location TEXT NULL,
			event_type TEXT DEFAULT 'casual',
			start_time TIMESTAMPTZ NOT NULL,
			finalized BOOLEAN NOT NULL DEFAULT FALSE,
			created_by TEXT NOT NULL,
			state TEXT NOT NULL DEFAULT 'UPCOMING',
			participants JSONB NOT NULL DEFAULT '[]'::jsonb,
			event_message_id TEXT,
			guild_id TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("failed creating legacy rounds table: %v", err)
	}

	roundID := "00000000-0000-0000-0000-000000000111"
	_, err = db.ExecContext(ctx, `
		INSERT INTO rounds (
			id, title, description, location, start_time, created_by, state, participants, guild_id
		) VALUES (
			?, 'Legacy Round', NULL, NULL, NOW() + INTERVAL '2 hour', 'user-admin', 'UPCOMING',
			'[{"user_id":"user-1"},{"user_id":"user-2"}]'::jsonb, 'guild-123'
		)
	`, roundID)
	if err != nil {
		t.Fatalf("failed inserting legacy round row: %v", err)
	}

	runSingleRoundMigration(t, ctx, db, "add_round_groups")

	var description, location, mode, teams string
	err = db.QueryRowContext(
		ctx,
		`SELECT description, location, mode, teams::text FROM rounds WHERE id = ?`,
		roundID,
	).Scan(&description, &location, &mode, &teams)
	if err != nil {
		t.Fatalf("failed querying migrated round row: %v", err)
	}

	if description != "" {
		t.Fatalf("expected description backfilled to empty string, got %q", description)
	}
	if location != "" {
		t.Fatalf("expected location backfilled to empty string, got %q", location)
	}
	if mode != "SINGLES" {
		t.Fatalf("expected mode to default to SINGLES, got %q", mode)
	}
	if teams != "[]" {
		t.Fatalf("expected teams to default to [], got %q", teams)
	}

	var groupCount int
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM round_groups WHERE round_id = ?`, roundID).Scan(&groupCount)
	if err != nil {
		t.Fatalf("failed counting round_groups rows: %v", err)
	}
	if groupCount != 2 {
		t.Fatalf("expected 2 round_groups rows, got %d", groupCount)
	}

	var memberCount int
	err = db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM round_group_members rgm
		 JOIN round_groups rg ON rgm.group_id = rg.id
		 WHERE rg.round_id = ?`,
		roundID,
	).Scan(&memberCount)
	if err != nil {
		t.Fatalf("failed counting round_group_members rows: %v", err)
	}
	if memberCount != 2 {
		t.Fatalf("expected 2 round_group_members rows, got %d", memberCount)
	}
}

func TestRoundMigration_EnsureDiscordEventID_EnforcesTextTypeAndIndex(t *testing.T) {
	db, ctx := setupIsolatedPostgresDB(t)

	_, err := db.ExecContext(ctx, `
		CREATE TABLE rounds (
			id UUID PRIMARY KEY,
			guild_id TEXT NOT NULL,
			discord_event_id VARCHAR(255)
		)
	`)
	if err != nil {
		t.Fatalf("failed creating rounds table: %v", err)
	}

	runSingleRoundMigration(t, ctx, db, "ensure_discord_event_id")

	var dataType string
	err = db.QueryRowContext(
		ctx,
		`SELECT data_type
		 FROM information_schema.columns
		 WHERE table_schema = 'public' AND table_name = 'rounds' AND column_name = 'discord_event_id'`,
	).Scan(&dataType)
	if err != nil {
		t.Fatalf("failed querying discord_event_id column metadata: %v", err)
	}
	if dataType != "text" {
		t.Fatalf("expected discord_event_id type text, got %q", dataType)
	}

	var indexCount int
	err = db.QueryRowContext(
		ctx,
		`SELECT COUNT(*)
		 FROM pg_indexes
		 WHERE schemaname = 'public' AND tablename = 'rounds' AND indexname = 'idx_rounds_discord_event_id'`,
	).Scan(&indexCount)
	if err != nil {
		t.Fatalf("failed querying discord_event_id index: %v", err)
	}
	if indexCount != 1 {
		t.Fatalf("expected idx_rounds_discord_event_id to exist exactly once, got %d", indexCount)
	}
}
