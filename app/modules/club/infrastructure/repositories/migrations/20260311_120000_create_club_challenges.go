package clubmigrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `
				CREATE TABLE IF NOT EXISTS club_challenges (
					uuid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					club_uuid UUID NOT NULL REFERENCES clubs(uuid) ON DELETE CASCADE,
					challenger_user_uuid UUID NOT NULL REFERENCES users(uuid) ON DELETE CASCADE,
					defender_user_uuid UUID NOT NULL REFERENCES users(uuid) ON DELETE CASCADE,
					status TEXT NOT NULL,
					original_challenger_tag INT,
					original_defender_tag INT,
					opened_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
					open_expires_at TIMESTAMPTZ,
					accepted_at TIMESTAMPTZ,
					accepted_expires_at TIMESTAMPTZ,
					completed_at TIMESTAMPTZ,
					hidden_at TIMESTAMPTZ,
					hidden_by_user_uuid UUID REFERENCES users(uuid) ON DELETE SET NULL,
					discord_guild_id VARCHAR(20),
					discord_channel_id VARCHAR(20),
					discord_message_id VARCHAR(20),
					created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
					updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
					CHECK (challenger_user_uuid <> defender_user_uuid)
				);

				CREATE INDEX IF NOT EXISTS idx_club_challenges_club_status ON club_challenges(club_uuid, status);
				CREATE INDEX IF NOT EXISTS idx_club_challenges_challenger_status ON club_challenges(club_uuid, challenger_user_uuid, status);
				CREATE INDEX IF NOT EXISTS idx_club_challenges_defender_status ON club_challenges(club_uuid, defender_user_uuid, status);

				-- Module migrations are intentionally order-independent in tests and fresh
				-- environments, so we cannot hard-reference rounds(id) here. The service
				-- validates round existence before links are created.
				CREATE TABLE IF NOT EXISTS club_challenge_round_links (
					uuid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					challenge_uuid UUID NOT NULL REFERENCES club_challenges(uuid) ON DELETE CASCADE,
					round_id UUID NOT NULL,
					linked_by_user_uuid UUID REFERENCES users(uuid) ON DELETE SET NULL,
					unlinked_by_user_uuid UUID REFERENCES users(uuid) ON DELETE SET NULL,
					linked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
					unlinked_at TIMESTAMPTZ
				);

				CREATE UNIQUE INDEX IF NOT EXISTS idx_club_challenge_round_links_active_challenge
					ON club_challenge_round_links(challenge_uuid)
					WHERE unlinked_at IS NULL;

				CREATE UNIQUE INDEX IF NOT EXISTS idx_club_challenge_round_links_active_round
					ON club_challenge_round_links(round_id)
					WHERE unlinked_at IS NULL;
			`); err != nil {
				return fmt.Errorf("failed to create club challenge tables: %w", err)
			}
			return nil
		})
	}, func(ctx context.Context, db *bun.DB) error {
		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `
				DROP TABLE IF EXISTS club_challenge_round_links;
				DROP TABLE IF EXISTS club_challenges;
			`); err != nil {
				return fmt.Errorf("failed to drop club challenge tables: %w", err)
			}
			return nil
		})
	})
}
