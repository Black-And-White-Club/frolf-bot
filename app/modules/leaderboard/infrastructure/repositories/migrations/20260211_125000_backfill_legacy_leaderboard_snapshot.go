package leaderboardmigrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Backfilling normalized leaderboard tables from legacy leaderboards snapshot...")

		var legacyTable sql.NullString
		if err := db.NewRaw("SELECT to_regclass('public.leaderboards')::text").Scan(ctx, &legacyTable); err != nil {
			return fmt.Errorf("check legacy leaderboards table: %w", err)
		}
		if !legacyTable.Valid || legacyTable.String == "" {
			fmt.Println("Legacy leaderboards table not found; skipping backfill.")
			return nil
		}

		// Ensure a default season row exists for guilds that do not currently have an active season.
		// We keep it inactive so teams can explicitly start a new season when ready.
		_, err := db.NewRaw(`
			WITH legacy_guilds AS (
				SELECT DISTINCT guild_id
				FROM leaderboards
				WHERE guild_id <> ''
			),
			guilds_without_active_season AS (
				SELECT lg.guild_id
				FROM legacy_guilds lg
				LEFT JOIN leaderboard_seasons s
					ON s.guild_id = lg.guild_id AND s.is_active = true
				WHERE s.id IS NULL
			)
			INSERT INTO leaderboard_seasons (guild_id, id, name, is_active)
			SELECT g.guild_id, 'default', 'Default Season', false
			FROM guilds_without_active_season g
			ON CONFLICT (guild_id, id) DO NOTHING
		`).Exec(ctx)
		if err != nil {
			return fmt.Errorf("ensure per-guild default seasons for backfill: %w", err)
		}

		// Backfill current tags into league_members.
		// For each guild, we prefer the active snapshot; if none is active, we use the latest row.
		// We never overwrite existing normalized rows.
		_, err = db.NewRaw(`
			WITH legacy_snapshots AS (
				SELECT DISTINCT ON (guild_id)
					guild_id,
					leaderboard_data
				FROM leaderboards
				WHERE guild_id <> ''
				ORDER BY guild_id, is_active DESC, id DESC
			),
			expanded AS (
				SELECT
					ls.guild_id,
					entry
				FROM legacy_snapshots ls,
				LATERAL jsonb_array_elements(
					CASE
						WHEN jsonb_typeof(ls.leaderboard_data) = 'array' THEN ls.leaderboard_data
						WHEN jsonb_typeof(ls.leaderboard_data) = 'object'
							AND ls.leaderboard_data ? 'leaderboard_data'
							AND jsonb_typeof(ls.leaderboard_data->'leaderboard_data') = 'array'
							THEN ls.leaderboard_data->'leaderboard_data'
						ELSE '[]'::jsonb
					END
				) AS entry
			),
			normalized AS (
				SELECT
					guild_id,
					NULLIF(BTRIM(entry->>'user_id'), '') AS user_id,
					CASE
						WHEN COALESCE(entry->>'tag_number', '') ~ '^-?[0-9]+$' THEN (entry->>'tag_number')::integer
						ELSE NULL
					END AS tag_number
				FROM expanded
			),
			ranked AS (
				SELECT
					guild_id,
					user_id,
					tag_number,
					ROW_NUMBER() OVER (PARTITION BY guild_id, user_id ORDER BY tag_number ASC, user_id ASC) AS user_rank,
					ROW_NUMBER() OVER (PARTITION BY guild_id, tag_number ORDER BY user_id ASC) AS tag_rank
				FROM normalized
				WHERE user_id IS NOT NULL
					AND tag_number IS NOT NULL
					AND tag_number > 0
			)
			INSERT INTO league_members (guild_id, member_id, current_tag, last_active_at, updated_at)
			SELECT guild_id, user_id, tag_number, now(), now()
			FROM ranked
			WHERE user_rank = 1
				AND tag_rank = 1
			ON CONFLICT (guild_id, member_id) DO NOTHING
		`).Exec(ctx)
		if err != nil {
			return fmt.Errorf("backfill league_members from legacy snapshot: %w", err)
		}

		// Backfill season standings so existing total_points/rounds_played are preserved.
		// We write into each guild's active season when present, otherwise into "default".
		// We never overwrite existing normalized standings.
		_, err = db.NewRaw(`
			WITH legacy_snapshots AS (
				SELECT DISTINCT ON (guild_id)
					guild_id,
					leaderboard_data
				FROM leaderboards
				WHERE guild_id <> ''
				ORDER BY guild_id, is_active DESC, id DESC
			),
			expanded AS (
				SELECT
					ls.guild_id,
					entry
				FROM legacy_snapshots ls,
				LATERAL jsonb_array_elements(
					CASE
						WHEN jsonb_typeof(ls.leaderboard_data) = 'array' THEN ls.leaderboard_data
						WHEN jsonb_typeof(ls.leaderboard_data) = 'object'
							AND ls.leaderboard_data ? 'leaderboard_data'
							AND jsonb_typeof(ls.leaderboard_data->'leaderboard_data') = 'array'
							THEN ls.leaderboard_data->'leaderboard_data'
						ELSE '[]'::jsonb
					END
				) AS entry
			),
			normalized AS (
				SELECT
					e.guild_id,
					NULLIF(BTRIM(e.entry->>'user_id'), '') AS user_id,
					CASE
						WHEN COALESCE(e.entry->>'tag_number', '') ~ '^-?[0-9]+$' THEN (e.entry->>'tag_number')::integer
						ELSE 0
					END AS tag_number,
					CASE
						WHEN COALESCE(e.entry->>'total_points', '') ~ '^-?[0-9]+$' THEN (e.entry->>'total_points')::integer
						ELSE 0
					END AS total_points,
					CASE
						WHEN COALESCE(e.entry->>'rounds_played', '') ~ '^-?[0-9]+$' THEN (e.entry->>'rounds_played')::integer
						ELSE 0
					END AS rounds_played
				FROM expanded e
			),
			target_season AS (
				SELECT
					g.guild_id,
					COALESCE(s.id, 'default') AS season_id
				FROM (SELECT DISTINCT guild_id FROM normalized WHERE guild_id <> '') g
				LEFT JOIN leaderboard_seasons s
					ON s.guild_id = g.guild_id
					AND s.is_active = true
			),
			aggregated AS (
				SELECT
					n.guild_id,
					n.user_id,
					MAX(n.tag_number) AS season_best_tag,
					MAX(n.total_points) AS total_points,
					MAX(n.rounds_played) AS rounds_played
				FROM normalized n
				WHERE n.user_id IS NOT NULL
				GROUP BY n.guild_id, n.user_id
			)
			INSERT INTO leaderboard_season_standings (
				guild_id,
				season_id,
				member_id,
				total_points,
				current_tier,
				season_best_tag,
				rounds_played,
				updated_at
			)
			SELECT
				a.guild_id,
				t.season_id,
				a.user_id,
				GREATEST(a.total_points, 0),
				'Bronze',
				GREATEST(a.season_best_tag, 0),
				GREATEST(a.rounds_played, 0),
				now()
			FROM aggregated a
			INNER JOIN target_season t
				ON t.guild_id = a.guild_id
			ON CONFLICT (guild_id, season_id, member_id) DO NOTHING
		`).Exec(ctx)
		if err != nil {
			return fmt.Errorf("backfill season standings from legacy snapshot: %w", err)
		}

		fmt.Println("Legacy leaderboard snapshot backfill completed successfully.")
		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Backfill rollback is a no-op (data-preserving migration).")
		return nil
	})
}
