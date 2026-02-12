package leaderboardmigrations

import (
	"context"
	"fmt"

	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Adding season scoping to leaderboard tables...")

		// 1. Create leaderboard_seasons table
		if _, err := db.NewCreateTable().Model((*leaderboarddb.Season)(nil)).IfNotExists().Exec(ctx); err != nil {
			return fmt.Errorf("create leaderboard_seasons: %w", err)
		}

		// 2. Insert default season
		_, err := db.NewRaw(`
			INSERT INTO leaderboard_seasons (guild_id, id, name, is_active)
			VALUES ('', 'default', 'Default Season', true)
			ON CONFLICT (guild_id, id) DO NOTHING
		`).Exec(ctx)
		if err != nil {
			return fmt.Errorf("insert default season: %w", err)
		}

		// 3. Add season_id to leaderboard_point_history
		_, err = db.NewRaw("ALTER TABLE leaderboard_point_history ADD COLUMN IF NOT EXISTS season_id VARCHAR(64) NOT NULL DEFAULT 'default'").Exec(ctx)
		if err != nil {
			return fmt.Errorf("add season_id to point_history: %w", err)
		}

		// 4. Add season_id to leaderboard_season_standings and update primary key
		_, err = db.NewRaw("ALTER TABLE leaderboard_season_standings ADD COLUMN IF NOT EXISTS season_id VARCHAR(64) NOT NULL DEFAULT 'default'").Exec(ctx)
		if err != nil {
			return fmt.Errorf("add season_id to season_standings: %w", err)
		}

		// Drop old primary key and create composite (season_id, member_id)
		_, err = db.NewRaw("ALTER TABLE leaderboard_season_standings DROP CONSTRAINT IF EXISTS leaderboard_season_standings_pkey").Exec(ctx)
		if err != nil {
			return fmt.Errorf("drop old standings pkey: %w", err)
		}
		_, err = db.NewRaw("ALTER TABLE leaderboard_season_standings ADD PRIMARY KEY (season_id, member_id)").Exec(ctx)
		if err != nil {
			return fmt.Errorf("add composite standings pkey: %w", err)
		}

		// 5. Add index for season queries
		_, err = db.NewRaw("CREATE INDEX IF NOT EXISTS idx_point_history_season_id ON leaderboard_point_history (season_id)").Exec(ctx)
		if err != nil {
			return fmt.Errorf("create point_history season index: %w", err)
		}
		_, err = db.NewRaw("CREATE INDEX IF NOT EXISTS idx_season_standings_season_id ON leaderboard_season_standings (season_id)").Exec(ctx)
		if err != nil {
			return fmt.Errorf("create season_standings season index: %w", err)
		}

		// Update upsert conflict target index for the new composite key
		_, err = db.NewRaw("CREATE UNIQUE INDEX IF NOT EXISTS idx_season_standings_season_member ON leaderboard_season_standings (season_id, member_id)").Exec(ctx)
		if err != nil {
			return fmt.Errorf("create season_standings unique index: %w", err)
		}

		fmt.Println("Season scoping added successfully!")
		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Removing season scoping from leaderboard tables...")

		// Restore original primary key
		_, _ = db.NewRaw("ALTER TABLE leaderboard_season_standings DROP CONSTRAINT IF EXISTS leaderboard_season_standings_pkey").Exec(ctx)
		_, _ = db.NewRaw("ALTER TABLE leaderboard_season_standings ADD PRIMARY KEY (member_id)").Exec(ctx)

		// Drop indexes
		_, _ = db.NewRaw("DROP INDEX IF EXISTS idx_point_history_season_id").Exec(ctx)
		_, _ = db.NewRaw("DROP INDEX IF EXISTS idx_season_standings_season_id").Exec(ctx)
		_, _ = db.NewRaw("DROP INDEX IF EXISTS idx_season_standings_season_member").Exec(ctx)

		// Drop columns
		_, _ = db.NewRaw("ALTER TABLE leaderboard_season_standings DROP COLUMN IF EXISTS season_id").Exec(ctx)
		_, _ = db.NewRaw("ALTER TABLE leaderboard_point_history DROP COLUMN IF EXISTS season_id").Exec(ctx)

		// Drop seasons table
		if _, err := db.NewDropTable().Model((*leaderboarddb.Season)(nil)).IfExists().Exec(ctx); err != nil {
			return err
		}

		fmt.Println("Season scoping removed successfully!")
		return nil
	})
}
