package leaderboardmigrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Adding normalized tag state tables (league_members, tag_history, round_outcomes)...")

		// 1. league_members: persistent tag state per guild member
		_, err := db.NewRaw(`
			CREATE TABLE IF NOT EXISTS league_members (
				guild_id        text        NOT NULL,
				member_id       text        NOT NULL,
				current_tag     integer     NULL,
				last_active_at  timestamptz NOT NULL DEFAULT now(),
				updated_at      timestamptz NOT NULL DEFAULT now(),
				PRIMARY KEY (guild_id, member_id)
			)
		`).Exec(ctx)
		if err != nil {
			return fmt.Errorf("create league_members: %w", err)
		}

		// Unique partial index: only one holder per tag per guild (excluding NULL and 0)
		_, err = db.NewRaw(`
			CREATE UNIQUE INDEX IF NOT EXISTS uq_league_members_tag_per_guild
			ON league_members (guild_id, current_tag)
			WHERE current_tag IS NOT NULL AND current_tag > 0
		`).Exec(ctx)
		if err != nil {
			return fmt.Errorf("create uq_league_members_tag_per_guild: %w", err)
		}

		_, err = db.NewRaw(`
			CREATE INDEX IF NOT EXISTS idx_league_members_guild_last_active
			ON league_members (guild_id, last_active_at DESC)
		`).Exec(ctx)
		if err != nil {
			return fmt.Errorf("create idx_league_members_guild_last_active: %w", err)
		}

		// 2. tag_history: immutable tag ledger
		_, err = db.NewRaw(`
			CREATE TABLE IF NOT EXISTS tag_history (
				id              bigserial    PRIMARY KEY,
				guild_id        text         NOT NULL,
				round_id        uuid         NULL,
				tag_number      integer      NOT NULL,
				old_member_id   text         NULL,
				new_member_id   text         NOT NULL,
				reason          text         NOT NULL,
				metadata        jsonb        NOT NULL DEFAULT '{}',
				created_at      timestamptz  NOT NULL DEFAULT now()
			)
		`).Exec(ctx)
		if err != nil {
			return fmt.Errorf("create tag_history: %w", err)
		}

		_, err = db.NewRaw(`
			CREATE INDEX IF NOT EXISTS idx_tag_history_guild_round
			ON tag_history (guild_id, round_id)
		`).Exec(ctx)
		if err != nil {
			return fmt.Errorf("create idx_tag_history_guild_round: %w", err)
		}

		_, err = db.NewRaw(`
			CREATE INDEX IF NOT EXISTS idx_tag_history_guild_tag_created
			ON tag_history (guild_id, tag_number, created_at DESC)
		`).Exec(ctx)
		if err != nil {
			return fmt.Errorf("create idx_tag_history_guild_tag_created: %w", err)
		}

		// 3. Add guild_id to leaderboard_seasons (tenant scoping)
		_, err = db.NewRaw(`
			ALTER TABLE leaderboard_seasons
			ADD COLUMN IF NOT EXISTS guild_id text NOT NULL DEFAULT ''
		`).Exec(ctx)
		if err != nil {
			return fmt.Errorf("add guild_id to leaderboard_seasons: %w", err)
		}

		// Drop old PK and recreate as composite
		_, err = db.NewRaw(`
			ALTER TABLE leaderboard_seasons DROP CONSTRAINT IF EXISTS leaderboard_seasons_pkey
		`).Exec(ctx)
		if err != nil {
			return fmt.Errorf("drop old seasons pkey: %w", err)
		}

		_, err = db.NewRaw(`
			ALTER TABLE leaderboard_seasons ADD PRIMARY KEY (guild_id, id)
		`).Exec(ctx)
		if err != nil {
			return fmt.Errorf("add composite seasons pkey: %w", err)
		}

		// Unique partial index: one active season per guild
		_, err = db.NewRaw(`
			CREATE UNIQUE INDEX IF NOT EXISTS uq_active_season_per_guild
			ON leaderboard_seasons (guild_id)
			WHERE is_active = true
		`).Exec(ctx)
		if err != nil {
			return fmt.Errorf("create uq_active_season_per_guild: %w", err)
		}

		// 4. Add guild_id to existing points/standings tables
		_, err = db.NewRaw(`
			ALTER TABLE leaderboard_season_standings
			ADD COLUMN IF NOT EXISTS guild_id text NOT NULL DEFAULT ''
		`).Exec(ctx)
		if err != nil {
			return fmt.Errorf("add guild_id to season_standings: %w", err)
		}

		// Update PK to include guild_id
		_, err = db.NewRaw(`
			ALTER TABLE leaderboard_season_standings DROP CONSTRAINT IF EXISTS leaderboard_season_standings_pkey
		`).Exec(ctx)
		if err != nil {
			return fmt.Errorf("drop old standings pkey: %w", err)
		}

		_, err = db.NewRaw(`
			ALTER TABLE leaderboard_season_standings ADD PRIMARY KEY (guild_id, season_id, member_id)
		`).Exec(ctx)
		if err != nil {
			return fmt.Errorf("add guild standings pkey: %w", err)
		}

		_, err = db.NewRaw(`
			CREATE INDEX IF NOT EXISTS idx_standings_guild_season_points
			ON leaderboard_season_standings (guild_id, season_id, total_points DESC)
		`).Exec(ctx)
		if err != nil {
			return fmt.Errorf("create idx_standings_guild_season_points: %w", err)
		}

		// Add guild_id to point_history
		_, err = db.NewRaw(`
			ALTER TABLE leaderboard_point_history
			ADD COLUMN IF NOT EXISTS guild_id text NOT NULL DEFAULT ''
		`).Exec(ctx)
		if err != nil {
			return fmt.Errorf("add guild_id to point_history: %w", err)
		}

		_, err = db.NewRaw(`
			CREATE INDEX IF NOT EXISTS idx_point_history_guild_round
			ON leaderboard_point_history (guild_id, round_id)
		`).Exec(ctx)
		if err != nil {
			return fmt.Errorf("create idx_point_history_guild_round: %w", err)
		}

		_, err = db.NewRaw(`
			CREATE INDEX IF NOT EXISTS idx_point_history_guild_member_created
			ON leaderboard_point_history (guild_id, member_id, created_at DESC)
		`).Exec(ctx)
		if err != nil {
			return fmt.Errorf("create idx_point_history_guild_member_created: %w", err)
		}

		// Add FK from season_standings to seasons
		_, err = db.NewRaw(`
			ALTER TABLE leaderboard_season_standings
			ADD CONSTRAINT IF NOT EXISTS fk_standings_season
			FOREIGN KEY (guild_id, season_id)
			REFERENCES leaderboard_seasons (guild_id, id)
		`).Exec(ctx)
		if err != nil {
			// Some PG versions don't support IF NOT EXISTS on constraints - ignore if already exists
			fmt.Printf("Note: FK constraint may already exist: %v\n", err)
		}

		// Add FK from point_history to seasons
		_, err = db.NewRaw(`
			ALTER TABLE leaderboard_point_history
			ADD CONSTRAINT IF NOT EXISTS fk_point_history_season
			FOREIGN KEY (guild_id, season_id)
			REFERENCES leaderboard_seasons (guild_id, id)
		`).Exec(ctx)
		if err != nil {
			fmt.Printf("Note: FK constraint may already exist: %v\n", err)
		}

		// 5. leaderboard_round_outcomes: idempotency and recalculation tracking
		_, err = db.NewRaw(`
			CREATE TABLE IF NOT EXISTS leaderboard_round_outcomes (
				guild_id         text         NOT NULL,
				round_id         uuid         NOT NULL,
				season_id        text         NULL,
				processing_hash  text         NOT NULL,
				processed_at     timestamptz  NOT NULL DEFAULT now(),
				PRIMARY KEY (guild_id, round_id)
			)
		`).Exec(ctx)
		if err != nil {
			return fmt.Errorf("create leaderboard_round_outcomes: %w", err)
		}

		// Add stats_json column to season_standings
		_, err = db.NewRaw(`
				ALTER TABLE leaderboard_season_standings
			ADD COLUMN IF NOT EXISTS stats_json jsonb NOT NULL DEFAULT '{}'
		`).Exec(ctx)
		if err != nil {
			return fmt.Errorf("add stats_json to season_standings: %w", err)
		}

		fmt.Println("Normalized tag state tables created successfully!")
		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Dropping normalized tag state tables...")

		_, _ = db.NewRaw("DROP TABLE IF EXISTS leaderboard_round_outcomes").Exec(ctx)
		_, _ = db.NewRaw("DROP TABLE IF EXISTS tag_history").Exec(ctx)
		_, _ = db.NewRaw("DROP TABLE IF EXISTS league_members").Exec(ctx)

		// Remove guild_id columns (reverse migration)
		_, _ = db.NewRaw("ALTER TABLE leaderboard_season_standings DROP COLUMN IF EXISTS guild_id").Exec(ctx)
		_, _ = db.NewRaw("ALTER TABLE leaderboard_season_standings DROP COLUMN IF EXISTS stats_json").Exec(ctx)
		_, _ = db.NewRaw("ALTER TABLE leaderboard_point_history DROP COLUMN IF EXISTS guild_id").Exec(ctx)
		_, _ = db.NewRaw("ALTER TABLE leaderboard_seasons DROP COLUMN IF EXISTS guild_id").Exec(ctx)

		// Restore original seasons PK
		_, _ = db.NewRaw("ALTER TABLE leaderboard_seasons DROP CONSTRAINT IF EXISTS leaderboard_seasons_pkey").Exec(ctx)
		_, _ = db.NewRaw("ALTER TABLE leaderboard_seasons ADD PRIMARY KEY (id)").Exec(ctx)

		// Restore original standings PK
		_, _ = db.NewRaw("ALTER TABLE leaderboard_season_standings DROP CONSTRAINT IF EXISTS leaderboard_season_standings_pkey").Exec(ctx)
		_, _ = db.NewRaw("ALTER TABLE leaderboard_season_standings ADD PRIMARY KEY (season_id, member_id)").Exec(ctx)

		// Drop new indexes
		_, _ = db.NewRaw("DROP INDEX IF EXISTS uq_active_season_per_guild").Exec(ctx)
		_, _ = db.NewRaw("DROP INDEX IF EXISTS idx_standings_guild_season_points").Exec(ctx)
		_, _ = db.NewRaw("DROP INDEX IF EXISTS idx_point_history_guild_round").Exec(ctx)
		_, _ = db.NewRaw("DROP INDEX IF EXISTS idx_point_history_guild_member_created").Exec(ctx)

		fmt.Println("Normalized tag state tables dropped successfully!")
		return nil
	})
}
