package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(
		func(ctx context.Context, db *bun.DB) error {
			fmt.Println("Creating guild_configs table...")
			_, err := db.ExecContext(ctx, `
				CREATE TABLE IF NOT EXISTS guild_configs (
					guild_id VARCHAR(20) PRIMARY KEY,
					created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
					is_active BOOLEAN NOT NULL DEFAULT true,
					signup_channel_id VARCHAR(20),
					signup_message_id VARCHAR(20),
					event_channel_id VARCHAR(20),
					leaderboard_channel_id VARCHAR(20),
					user_role_id VARCHAR(20),
					editor_role_id VARCHAR(20),
					admin_role_id VARCHAR(20),
					signup_emoji VARCHAR(10) NOT NULL DEFAULT '🐍',
					auto_setup_completed BOOLEAN NOT NULL DEFAULT false,
					setup_completed_at TIMESTAMPTZ,
					subscription_tier VARCHAR(20),
					subscription_expires_at TIMESTAMPTZ,
					is_trial BOOLEAN,
					trial_expires_at TIMESTAMPTZ,
					max_concurrent_rounds INTEGER,
					max_participants_per_round INTEGER,
					commands_per_minute INTEGER,
					rounds_per_day INTEGER,
					custom_leaderboards_enabled BOOLEAN
				);
			`)
			if err != nil {
				return fmt.Errorf("failed to create guild_configs table: %w", err)
			}
			fmt.Println("guild_configs table created successfully!")
			return nil
		},
		func(ctx context.Context, db *bun.DB) error {
			fmt.Println("Dropping guild_configs table...")
			_, err := db.ExecContext(ctx, `DROP TABLE IF EXISTS guild_configs CASCADE;`)
			if err != nil {
				return fmt.Errorf("failed to drop guild_configs table: %w", err)
			}
			fmt.Println("guild_configs table dropped successfully!")
			return nil
		},
	)
}
