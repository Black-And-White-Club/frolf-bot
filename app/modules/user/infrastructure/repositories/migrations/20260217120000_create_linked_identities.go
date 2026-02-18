package usermigrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Creating linked_identities table and backfilling from users.user_id...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `
				CREATE TABLE IF NOT EXISTS linked_identities (
					id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					user_uuid    UUID NOT NULL REFERENCES users(uuid) ON DELETE CASCADE,
					provider     TEXT NOT NULL,
					provider_id  TEXT NOT NULL,
					display_name TEXT,
					linked_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
					UNIQUE (provider, provider_id)
				);
				CREATE INDEX IF NOT EXISTS idx_linked_identities_user_uuid ON linked_identities (user_uuid);
			`); err != nil {
				return fmt.Errorf("failed to create linked_identities table: %w", err)
			}

			// Backfill: migrate existing discord user_id values from users table.
			// This populates linked_identities for all users that already have a
			// Discord ID stored in users.user_id, making them discoverable via
			// the new FindUserByLinkedIdentity repo method.
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO linked_identities (user_uuid, provider, provider_id, display_name, linked_at)
				SELECT uuid, 'discord', user_id::TEXT, display_name, created_at
				FROM users
				WHERE user_id IS NOT NULL
				ON CONFLICT (provider, provider_id) DO NOTHING;
			`); err != nil {
				return fmt.Errorf("failed to backfill linked_identities from users: %w", err)
			}

			return nil
		})
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Rolling back linked_identities table...")

		return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS linked_identities;`); err != nil {
				return fmt.Errorf("failed to drop linked_identities: %w", err)
			}
			return nil
		})
	})
}
