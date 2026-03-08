package testutils

import (
	"context"
	"testing"

	"github.com/uptrace/bun"

	usermigrations "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories/migrations"
)

func assertColumnExists(t *testing.T, ctx context.Context, db *bun.DB, tableName, columnName string) {
	t.Helper()

	var count int
	err := db.QueryRowContext(
		ctx,
		`SELECT COUNT(*)
		 FROM information_schema.columns
		 WHERE table_schema = 'public' AND table_name = ? AND column_name = ?`,
		tableName,
		columnName,
	).Scan(&count)
	if err != nil {
		t.Fatalf("failed querying column existence for %s.%s: %v", tableName, columnName, err)
	}
	if count != 1 {
		t.Fatalf("expected column %s.%s to exist exactly once, got %d", tableName, columnName, count)
	}
}

func assertColumnMissing(t *testing.T, ctx context.Context, db *bun.DB, tableName, columnName string) {
	t.Helper()

	var count int
	err := db.QueryRowContext(
		ctx,
		`SELECT COUNT(*)
		 FROM information_schema.columns
		 WHERE table_schema = 'public' AND table_name = ? AND column_name = ?`,
		tableName,
		columnName,
	).Scan(&count)
	if err != nil {
		t.Fatalf("failed querying column existence for %s.%s: %v", tableName, columnName, err)
	}
	if count != 0 {
		t.Fatalf("expected column %s.%s to be absent, got count=%d", tableName, columnName, count)
	}
}

func TestUserMigrations_TargetedInvariants(t *testing.T) {
	tests := []struct {
		name              string
		migrationContains string
		setup             func(t *testing.T, ctx context.Context, db *bun.DB)
		assertions        func(t *testing.T, ctx context.Context, db *bun.DB)
	}{
		{
			name:              "split users to users plus guild_memberships",
			migrationContains: "split_users_guild_memberships",
			setup: func(t *testing.T, ctx context.Context, db *bun.DB) {
				t.Helper()

				_, err := db.ExecContext(ctx, `
					CREATE TABLE users (
						id BIGSERIAL PRIMARY KEY,
						user_id TEXT NOT NULL,
						guild_id TEXT NOT NULL,
						role TEXT NOT NULL DEFAULT 'User',
						udisc_username TEXT,
						udisc_name TEXT,
						created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
						UNIQUE(user_id, guild_id)
					)
				`)
				if err != nil {
					t.Fatalf("failed creating legacy users table: %v", err)
				}

				_, err = db.ExecContext(ctx, `
					INSERT INTO users (user_id, guild_id, role, udisc_username, udisc_name, created_at)
					VALUES
					('user-1', 'guild-a', 'Admin', 'u1', 'User One', NOW() - INTERVAL '3 day'),
					('user-1', 'guild-b', 'Player', 'u1', 'User One', NOW() - INTERVAL '2 day'),
					('user-2', 'guild-a', 'Player', 'u2', 'User Two', NOW() - INTERVAL '1 day')
				`)
				if err != nil {
					t.Fatalf("failed inserting legacy users rows: %v", err)
				}
			},
			assertions: func(t *testing.T, ctx context.Context, db *bun.DB) {
				t.Helper()

				assertTableExists(t, ctx, db, "users")
				assertTableExists(t, ctx, db, "guild_memberships")
				assertColumnExists(t, ctx, db, "users", "updated_at")
				assertColumnMissing(t, ctx, db, "users", "guild_id")
				assertIndexExists(t, ctx, db, "guild_memberships", "idx_guild_memberships_user_id")
				assertIndexExists(t, ctx, db, "guild_memberships", "idx_guild_memberships_guild_id")

				var usersCount int
				if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&usersCount); err != nil {
					t.Fatalf("failed counting users rows: %v", err)
				}
				if usersCount != 2 {
					t.Fatalf("expected 2 users after dedupe, got %d", usersCount)
				}

				var membershipsCount int
				if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM guild_memberships`).Scan(&membershipsCount); err != nil {
					t.Fatalf("failed counting guild_memberships rows: %v", err)
				}
				if membershipsCount != 3 {
					t.Fatalf("expected 3 guild_memberships rows, got %d", membershipsCount)
				}
			},
		},
		{
			name:              "setup club memberships adds and backfills table",
			migrationContains: "setup_club_memberships",
			setup: func(t *testing.T, ctx context.Context, db *bun.DB) {
				t.Helper()

				_, err := db.ExecContext(ctx, `CREATE EXTENSION IF NOT EXISTS pgcrypto`)
				if err != nil {
					t.Fatalf("failed enabling pgcrypto extension: %v", err)
				}

				_, err = db.ExecContext(ctx, `
					CREATE TABLE users (
						id BIGSERIAL PRIMARY KEY,
						user_id TEXT UNIQUE NOT NULL,
						display_name TEXT,
						created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
						updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
					);
					CREATE TABLE guild_memberships (
						id BIGSERIAL PRIMARY KEY,
						user_id TEXT NOT NULL,
						guild_id TEXT NOT NULL,
						role TEXT NOT NULL,
						joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
						UNIQUE(user_id, guild_id)
					);
					CREATE TABLE guild_configs (
						guild_id TEXT PRIMARY KEY,
						uuid UUID UNIQUE NOT NULL,
						created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
						updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
					)
				`)
				if err != nil {
					t.Fatalf("failed creating prerequisite tables: %v", err)
				}

				_, err = db.ExecContext(ctx, `
					INSERT INTO users (user_id, display_name) VALUES ('user-9', 'Player Nine');
					INSERT INTO guild_configs (guild_id, uuid) VALUES ('guild-9', '00000000-0000-0000-0000-000000000901');
					INSERT INTO guild_memberships (user_id, guild_id, role) VALUES ('user-9', 'guild-9', 'editor')
				`)
				if err != nil {
					t.Fatalf("failed inserting prerequisite rows: %v", err)
				}
			},
			assertions: func(t *testing.T, ctx context.Context, db *bun.DB) {
				t.Helper()

				assertColumnExists(t, ctx, db, "users", "uuid")
				assertTableExists(t, ctx, db, "club_memberships")
				assertIndexExists(t, ctx, db, "guild_memberships", "idx_guild_memberships_guild_user_id")

				var count int
				err := db.QueryRowContext(
					ctx,
					`SELECT COUNT(*) FROM club_memberships WHERE external_id = 'user-9'`,
				).Scan(&count)
				if err != nil {
					t.Fatalf("failed counting club_memberships backfill rows: %v", err)
				}
				if count != 1 {
					t.Fatalf("expected 1 backfilled club_memberships row, got %d", count)
				}
			},
		},
		{
			name:              "create linked identities backfills discord identities",
			migrationContains: "create_linked_identities",
			setup: func(t *testing.T, ctx context.Context, db *bun.DB) {
				t.Helper()

				_, err := db.ExecContext(ctx, `
					CREATE TABLE users (
						uuid UUID PRIMARY KEY,
						user_id TEXT,
						display_name TEXT,
						created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
					)
				`)
				if err != nil {
					t.Fatalf("failed creating users table: %v", err)
				}

				_, err = db.ExecContext(ctx, `
					INSERT INTO users (uuid, user_id, display_name, created_at) VALUES
					('00000000-0000-0000-0000-000000000111', 'discord-111', 'Discord One', NOW() - INTERVAL '2 day'),
					('00000000-0000-0000-0000-000000000222', NULL, 'No Discord', NOW() - INTERVAL '1 day')
				`)
				if err != nil {
					t.Fatalf("failed inserting users rows: %v", err)
				}
			},
			assertions: func(t *testing.T, ctx context.Context, db *bun.DB) {
				t.Helper()

				assertTableExists(t, ctx, db, "linked_identities")
				assertIndexExists(t, ctx, db, "linked_identities", "idx_linked_identities_user_uuid")

				var count int
				err := db.QueryRowContext(
					ctx,
					`SELECT COUNT(*) FROM linked_identities WHERE provider = 'discord'`,
				).Scan(&count)
				if err != nil {
					t.Fatalf("failed counting linked identities rows: %v", err)
				}
				if count != 1 {
					t.Fatalf("expected 1 backfilled linked identity row, got %d", count)
				}
			},
		},
		{
			name:              "rename magic link token to token_hash preserves data",
			migrationContains: "rename_magic_link_token_to_token_hash",
			setup: func(t *testing.T, ctx context.Context, db *bun.DB) {
				t.Helper()

				_, err := db.ExecContext(ctx, `
					CREATE TABLE magic_links (
						token VARCHAR(64) PRIMARY KEY,
						created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
					)
				`)
				if err != nil {
					t.Fatalf("failed creating magic_links table: %v", err)
				}

				_, err = db.ExecContext(ctx, `INSERT INTO magic_links (token) VALUES ('abc123')`)
				if err != nil {
					t.Fatalf("failed inserting magic_links row: %v", err)
				}
			},
			assertions: func(t *testing.T, ctx context.Context, db *bun.DB) {
				t.Helper()

				assertColumnExists(t, ctx, db, "magic_links", "token_hash")
				assertColumnMissing(t, ctx, db, "magic_links", "token")

				var tokenHash string
				err := db.QueryRowContext(ctx, `SELECT token_hash FROM magic_links LIMIT 1`).Scan(&tokenHash)
				if err != nil {
					t.Fatalf("failed querying token_hash value: %v", err)
				}
				if tokenHash != "abc123" {
					t.Fatalf("expected token_hash to preserve value abc123, got %q", tokenHash)
				}
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			db, ctx := setupIsolatedPostgresDB(t)
			tc.setup(t, ctx, db)

			runSingleModuleMigration(t, ctx, db, usermigrations.Migrations, tc.migrationContains)
			tc.assertions(t, ctx, db)
		})
	}
}
