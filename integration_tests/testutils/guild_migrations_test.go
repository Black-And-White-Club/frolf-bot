package testutils

import (
	"context"
	"testing"

	"github.com/uptrace/bun"

	guildmigrations "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/infrastructure/repositories/migrations"
)

func TestGuildMigrations_TargetedInvariants(t *testing.T) {
	tests := []struct {
		name              string
		migrationContains string
		setup             func(t *testing.T, ctx context.Context, db *bun.DB)
		assertions        func(t *testing.T, ctx context.Context, db *bun.DB)
	}{
		{
			name:              "add resource state and deletion status columns",
			migrationContains: "add_resource_state_to_guild_configs",
			setup: func(t *testing.T, ctx context.Context, db *bun.DB) {
				t.Helper()

				_, err := db.ExecContext(ctx, `
					CREATE TABLE guild_configs (
						guild_id VARCHAR(20) PRIMARY KEY,
						signup_emoji VARCHAR(10) NOT NULL DEFAULT 'x'
					);
					INSERT INTO guild_configs (guild_id) VALUES ('guild-11');
				`)
				if err != nil {
					t.Fatalf("failed creating baseline guild_configs table: %v", err)
				}
			},
			assertions: func(t *testing.T, ctx context.Context, db *bun.DB) {
				t.Helper()

				assertColumnExists(t, ctx, db, "guild_configs", "resource_state")
				assertColumnExists(t, ctx, db, "guild_configs", "deletion_status")

				var deletionStatus string
				err := db.QueryRowContext(
					ctx,
					`SELECT deletion_status FROM guild_configs WHERE guild_id = 'guild-11'`,
				).Scan(&deletionStatus)
				if err != nil {
					t.Fatalf("failed reading deletion_status default: %v", err)
				}
				if deletionStatus != "none" {
					t.Fatalf("expected deletion_status default 'none', got %q", deletionStatus)
				}
			},
		},
		{
			name:              "add uuid column backfills existing rows",
			migrationContains: "add_uuid_to_guild_configs",
			setup: func(t *testing.T, ctx context.Context, db *bun.DB) {
				t.Helper()

				_, err := db.ExecContext(ctx, `
					CREATE TABLE guild_configs (
						guild_id VARCHAR(20) PRIMARY KEY,
						created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
						updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
						is_active BOOLEAN NOT NULL DEFAULT true,
						signup_emoji VARCHAR(10) NOT NULL DEFAULT 'snake',
						deletion_status VARCHAR(20) NOT NULL DEFAULT 'none'
					);
					INSERT INTO guild_configs (guild_id) VALUES ('guild-22');
				`)
				if err != nil {
					t.Fatalf("failed creating pre-uuid guild_configs table: %v", err)
				}
			},
			assertions: func(t *testing.T, ctx context.Context, db *bun.DB) {
				t.Helper()

				assertColumnExists(t, ctx, db, "guild_configs", "uuid")

				var uuidValue string
				err := db.QueryRowContext(
					ctx,
					`SELECT uuid::text FROM guild_configs WHERE guild_id = 'guild-22'`,
				).Scan(&uuidValue)
				if err != nil {
					t.Fatalf("failed querying generated uuid value: %v", err)
				}
				if uuidValue == "" {
					t.Fatal("expected non-empty generated uuid value")
				}

				var uniqueIndexCount int
				err = db.QueryRowContext(
					ctx,
					`SELECT COUNT(*)
					 FROM pg_indexes
					 WHERE schemaname = 'public'
					   AND tablename = 'guild_configs'
					   AND indexdef ILIKE '%UNIQUE%'
					   AND indexdef ILIKE '%(uuid)%'`,
				).Scan(&uniqueIndexCount)
				if err != nil {
					t.Fatalf("failed checking unique uuid index: %v", err)
				}
				if uniqueIndexCount < 1 {
					t.Fatalf("expected unique uuid index on guild_configs, got %d", uniqueIndexCount)
				}
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			db, ctx := setupIsolatedPostgresDB(t)
			tc.setup(t, ctx, db)

			runSingleModuleMigration(t, ctx, db, guildmigrations.Migrations, tc.migrationContains)
			tc.assertions(t, ctx, db)
		})
	}
}

func TestGuildMigrations_RunAllUp_SmokeAndSchemaInvariants(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "default"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			db, ctx := setupIsolatedPostgresDB(t)
			runAllModuleMigrations(t, ctx, db, guildmigrations.Migrations)

			assertTableExists(t, ctx, db, "guild_configs")
			assertColumnExists(t, ctx, db, "guild_configs", "resource_state")
			assertColumnExists(t, ctx, db, "guild_configs", "deletion_status")
			assertColumnExists(t, ctx, db, "guild_configs", "uuid")

			_, err := db.ExecContext(ctx, `INSERT INTO guild_configs (guild_id) VALUES ('guild-smoke')`)
			if err != nil {
				t.Fatalf("failed inserting smoke guild_configs row: %v", err)
			}

			var deletionStatus string
			err = db.QueryRowContext(
				ctx,
				`SELECT deletion_status FROM guild_configs WHERE guild_id = 'guild-smoke'`,
			).Scan(&deletionStatus)
			if err != nil {
				t.Fatalf("failed reading smoke row deletion_status: %v", err)
			}
			if deletionStatus != "none" {
				t.Fatalf("expected deletion_status='none' after all migrations, got %q", deletionStatus)
			}
		})
	}
}
