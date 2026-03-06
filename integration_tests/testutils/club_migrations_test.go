package testutils

import (
	"context"
	"testing"

	"github.com/uptrace/bun"

	clubmigrations "github.com/Black-And-White-Club/frolf-bot/app/modules/club/infrastructure/repositories/migrations"
)

func TestClubMigrations_TargetedInvariants(t *testing.T) {
	tests := []struct {
		name              string
		migrationContains string
		setup             func(t *testing.T, ctx context.Context, db *bun.DB)
		assertions        func(t *testing.T, ctx context.Context, db *bun.DB)
	}{
		{
			name:              "create clubs table backfills and rewires membership fk",
			migrationContains: "create_clubs_table",
			setup: func(t *testing.T, ctx context.Context, db *bun.DB) {
				t.Helper()

				_, err := db.ExecContext(ctx, `
					CREATE EXTENSION IF NOT EXISTS pgcrypto;
					CREATE TABLE guild_configs (
						guild_id VARCHAR(20) PRIMARY KEY,
						uuid UUID UNIQUE NOT NULL,
						created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
						updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
					);
					CREATE TABLE club_memberships (
						id BIGSERIAL PRIMARY KEY,
						club_uuid UUID NOT NULL,
						external_id TEXT NOT NULL,
						provider TEXT NOT NULL DEFAULT 'discord',
						role TEXT NOT NULL DEFAULT 'player'
					);
					ALTER TABLE club_memberships
						ADD CONSTRAINT club_memberships_club_uuid_fkey
						FOREIGN KEY (club_uuid) REFERENCES guild_configs(uuid) ON DELETE CASCADE;
					INSERT INTO guild_configs (guild_id, uuid)
					VALUES ('guild-90', '00000000-0000-0000-0000-000000000090');
					INSERT INTO club_memberships (club_uuid, external_id, provider, role)
					VALUES ('00000000-0000-0000-0000-000000000090', 'user-90', 'discord', 'editor');
				`)
				if err != nil {
					t.Fatalf("failed creating club migration prerequisites: %v", err)
				}
			},
			assertions: func(t *testing.T, ctx context.Context, db *bun.DB) {
				t.Helper()

				assertTableExists(t, ctx, db, "clubs")
				assertIndexExists(t, ctx, db, "clubs", "idx_clubs_discord_guild_id")
				assertForeignKeyConstraintReferences(t, ctx, db, "fk_club_memberships_club", "clubs")

				var count int
				err := db.QueryRowContext(
					ctx,
					`SELECT COUNT(*)
					 FROM clubs
					 WHERE discord_guild_id = 'guild-90'
					   AND uuid = '00000000-0000-0000-0000-000000000090'::uuid`,
				).Scan(&count)
				if err != nil {
					t.Fatalf("failed querying clubs backfill row: %v", err)
				}
				if count != 1 {
					t.Fatalf("expected exactly one backfilled clubs row, got %d", count)
				}
			},
		},
		{
			name:              "create club invites table and indexes",
			migrationContains: "create_club_invites",
			setup: func(t *testing.T, ctx context.Context, db *bun.DB) {
				t.Helper()

				_, err := db.ExecContext(ctx, `
					CREATE EXTENSION IF NOT EXISTS pgcrypto;
					CREATE TABLE clubs (
						uuid UUID PRIMARY KEY DEFAULT gen_random_uuid()
					);
					CREATE TABLE users (
						uuid UUID PRIMARY KEY DEFAULT gen_random_uuid()
					);
					INSERT INTO clubs (uuid) VALUES ('00000000-0000-0000-0000-000000000301');
					INSERT INTO users (uuid) VALUES ('00000000-0000-0000-0000-000000000302');
				`)
				if err != nil {
					t.Fatalf("failed creating club_invites prerequisites: %v", err)
				}
			},
			assertions: func(t *testing.T, ctx context.Context, db *bun.DB) {
				t.Helper()

				assertTableExists(t, ctx, db, "club_invites")
				assertIndexExists(t, ctx, db, "club_invites", "idx_club_invites_club_uuid")
				assertIndexExists(t, ctx, db, "club_invites", "idx_club_invites_code")

				_, err := db.ExecContext(
					ctx,
					`INSERT INTO club_invites (club_uuid, created_by, code, role)
					 VALUES ('00000000-0000-0000-0000-000000000301', '00000000-0000-0000-0000-000000000302', 'INVITE123456', 'player')`,
				)
				if err != nil {
					t.Fatalf("failed inserting club_invites row: %v", err)
				}
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			db, ctx := setupIsolatedPostgresDB(t)
			tc.setup(t, ctx, db)

			runSingleModuleMigration(t, ctx, db, clubmigrations.Migrations, tc.migrationContains)
			tc.assertions(t, ctx, db)
		})
	}
}

func TestClubMigrations_RunAllUp_SmokeAndSchemaInvariants(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "default"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			db, ctx := setupIsolatedPostgresDB(t)

			_, err := db.ExecContext(ctx, `
				CREATE EXTENSION IF NOT EXISTS pgcrypto;
				CREATE TABLE guild_configs (
					guild_id VARCHAR(20) PRIMARY KEY,
					uuid UUID UNIQUE NOT NULL,
					created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
					updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
				);
				CREATE TABLE club_memberships (
					id BIGSERIAL PRIMARY KEY,
					club_uuid UUID NOT NULL,
					external_id TEXT NOT NULL,
					provider TEXT NOT NULL DEFAULT 'discord',
					role TEXT NOT NULL DEFAULT 'player'
				);
				ALTER TABLE club_memberships
					ADD CONSTRAINT club_memberships_club_uuid_fkey
					FOREIGN KEY (club_uuid) REFERENCES guild_configs(uuid) ON DELETE CASCADE;
				CREATE TABLE users (
					uuid UUID PRIMARY KEY DEFAULT gen_random_uuid()
				);
				INSERT INTO guild_configs (guild_id, uuid)
				VALUES ('guild-smoke', '00000000-0000-0000-0000-000000000901');
				INSERT INTO club_memberships (club_uuid, external_id, provider, role)
				VALUES ('00000000-0000-0000-0000-000000000901', 'user-smoke', 'discord', 'player');
			`)
			if err != nil {
				t.Fatalf("failed preparing club smoke prerequisites: %v", err)
			}

			runAllModuleMigrations(t, ctx, db, clubmigrations.Migrations)

			assertTableExists(t, ctx, db, "clubs")
			assertTableExists(t, ctx, db, "club_invites")
			assertForeignKeyConstraintReferences(t, ctx, db, "fk_club_memberships_club", "clubs")

			var count int
			err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM clubs WHERE discord_guild_id = 'guild-smoke'`).Scan(&count)
			if err != nil {
				t.Fatalf("failed counting smoke clubs row: %v", err)
			}
			if count != 1 {
				t.Fatalf("expected 1 clubs row for guild-smoke, got %d", count)
			}
		})
	}
}
