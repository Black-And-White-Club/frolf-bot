package testutils

import (
	"context"
	"testing"

	"github.com/uptrace/bun"

	leaderboardmigrations "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories/migrations"
)

func TestLeaderboardMigrations_TargetedInvariants(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T, ctx context.Context, db *bun.DB)
		assertions func(t *testing.T, ctx context.Context, db *bun.DB)
	}{
		{
			name: "season scoping and normalized schema invariants hold after migration chain",
			setup: func(t *testing.T, ctx context.Context, db *bun.DB) {
				t.Helper()
				_ = ctx
				_ = db
			},
			assertions: func(t *testing.T, ctx context.Context, db *bun.DB) {
				t.Helper()

				assertTableExists(t, ctx, db, "leaderboard_seasons")
				assertColumnExists(t, ctx, db, "leaderboard_point_history", "season_id")
				assertColumnExists(t, ctx, db, "leaderboard_point_history", "guild_id")
				assertColumnExists(t, ctx, db, "leaderboard_season_standings", "season_id")
				assertColumnExists(t, ctx, db, "leaderboard_season_standings", "guild_id")
				assertPrimaryKeyColumns(
					t,
					ctx,
					db,
					"leaderboard_season_standings",
					[]string{"guild_id", "season_id", "member_id"},
				)
				assertIndexExists(t, ctx, db, "leaderboard_point_history", "idx_point_history_season_id")
				assertIndexExists(t, ctx, db, "leaderboard_season_standings", "idx_season_standings_season_id")
				assertTableExists(t, ctx, db, "league_members")
				assertTableExists(t, ctx, db, "tag_history")
				assertTableExists(t, ctx, db, "leaderboard_round_outcomes")
			},
		},
		{
			name: "legacy snapshot data backfills normalized tables",
			setup: func(t *testing.T, ctx context.Context, db *bun.DB) {
				t.Helper()

				_, err := db.ExecContext(ctx, `
					CREATE TABLE IF NOT EXISTS leaderboards (
						id BIGSERIAL PRIMARY KEY,
						leaderboard_data JSONB NOT NULL,
						is_active BOOLEAN NOT NULL DEFAULT true,
						update_source TEXT,
						update_id UUID,
						guild_id TEXT NOT NULL
					);
					INSERT INTO leaderboards (leaderboard_data, is_active, guild_id) VALUES (
						'[
							{"tag_number":1,"user_id":"user-a","total_points":1234,"rounds_played":14},
							{"tag_number":2,"user_id":"user-b","total_points":900,"rounds_played":10}
						]'::jsonb,
						true,
						'guild-77'
					)
				`)
				if err != nil {
					t.Fatalf("failed seeding legacy leaderboard snapshot: %v", err)
				}
			},
			assertions: func(t *testing.T, ctx context.Context, db *bun.DB) {
				t.Helper()

				var leagueMembersCount int
				err := db.QueryRowContext(
					ctx,
					`SELECT COUNT(*) FROM league_members WHERE guild_id = 'guild-77'`,
				).Scan(&leagueMembersCount)
				if err != nil {
					t.Fatalf("failed counting backfilled league_members rows: %v", err)
				}
				if leagueMembersCount != 2 {
					t.Fatalf("expected 2 backfilled league_members rows, got %d", leagueMembersCount)
				}

				var standingsCount int
				err = db.QueryRowContext(
					ctx,
					`SELECT COUNT(*)
					 FROM leaderboard_season_standings
					 WHERE guild_id = 'guild-77' AND season_id = 'default'`,
				).Scan(&standingsCount)
				if err != nil {
					t.Fatalf("failed counting backfilled standings rows: %v", err)
				}
				if standingsCount != 2 {
					t.Fatalf("expected 2 backfilled standings rows, got %d", standingsCount)
				}

				var seasonCount int
				err = db.QueryRowContext(
					ctx,
					`SELECT COUNT(*) FROM leaderboard_seasons WHERE guild_id = 'guild-77' AND id = 'default'`,
				).Scan(&seasonCount)
				if err != nil {
					t.Fatalf("failed counting inserted default season row: %v", err)
				}
				if seasonCount != 1 {
					t.Fatalf("expected 1 inserted default season for guild-77, got %d", seasonCount)
				}
			},
		},
		{
			name: "legacy leaderboards table is dropped by end of migration chain",
			setup: func(t *testing.T, ctx context.Context, db *bun.DB) {
				t.Helper()
				_ = ctx
				_ = db
			},
			assertions: func(t *testing.T, ctx context.Context, db *bun.DB) {
				t.Helper()
				assertTableMissing(t, ctx, db, "leaderboards")
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			db, ctx := setupIsolatedPostgresDB(t)
			tc.setup(t, ctx, db)

			runAllModuleMigrations(t, ctx, db, leaderboardmigrations.Migrations)
			tc.assertions(t, ctx, db)
		})
	}
}

func TestLeaderboardMigrations_RunAllUp_SmokeAndSchemaInvariants(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "default"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			db, ctx := setupIsolatedPostgresDB(t)
			runAllModuleMigrations(t, ctx, db, leaderboardmigrations.Migrations)

			assertTableExists(t, ctx, db, "leaderboard_seasons")
			assertTableExists(t, ctx, db, "leaderboard_point_history")
			assertTableExists(t, ctx, db, "leaderboard_season_standings")
			assertTableExists(t, ctx, db, "league_members")
			assertTableExists(t, ctx, db, "tag_history")
			assertTableExists(t, ctx, db, "leaderboard_round_outcomes")
			assertTableMissing(t, ctx, db, "leaderboards")

			assertPrimaryKeyColumns(
				t,
				ctx,
				db,
				"leaderboard_season_standings",
				[]string{"guild_id", "season_id", "member_id"},
			)
			assertIndexExists(t, ctx, db, "tag_history", "idx_tag_history_guild_tag_timeline")
			assertIndexExists(t, ctx, db, "tag_history", "idx_tag_history_new_member_timeline")
			assertIndexExists(t, ctx, db, "tag_history", "idx_tag_history_old_member_timeline")
			assertIndexExists(t, ctx, db, "tag_history", "idx_tag_history_guild_created")
		})
	}
}
