package leaderboardmigrations

import (
	"context"
	"fmt"

	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Creating leaderboard_point_history and leaderboard_season_standings tables...")

		if _, err := db.NewCreateTable().Model((*leaderboarddb.PointHistory)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}

		if _, err := db.NewCreateTable().Model((*leaderboarddb.SeasonStanding)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}

		// Create indexes manually if needed, though Bun might handle some via tags
		_, err := db.NewRaw("CREATE INDEX IF NOT EXISTS idx_point_history_member_id ON leaderboard_point_history (member_id)").Exec(ctx)
		if err != nil {
			return err
		}
		_, err = db.NewRaw("CREATE INDEX IF NOT EXISTS idx_point_history_round_id ON leaderboard_point_history (round_id)").Exec(ctx)
		if err != nil {
			return err
		}
		_, err = db.NewRaw("CREATE INDEX IF NOT EXISTS idx_season_standings_total_points ON leaderboard_season_standings (total_points DESC)").Exec(ctx)
		if err != nil {
			return err
		}
		// Composite index for rollback and history queries
		_, err = db.NewRaw("CREATE INDEX IF NOT EXISTS idx_point_history_member_round ON leaderboard_point_history (member_id, round_id)").Exec(ctx)
		if err != nil {
			return err
		}

		fmt.Println("Leaderboard tables created successfully!")
		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Dropping leaderboard_point_history and leaderboard_season_standings tables...")

		if _, err := db.NewDropTable().Model((*leaderboarddb.SeasonStanding)(nil)).IfExists().Exec(ctx); err != nil {
			return err
		}

		if _, err := db.NewDropTable().Model((*leaderboarddb.PointHistory)(nil)).IfExists().Exec(ctx); err != nil {
			return err
		}

		fmt.Println("Leaderboard tables dropped successfully!")
		return nil
	})
}
