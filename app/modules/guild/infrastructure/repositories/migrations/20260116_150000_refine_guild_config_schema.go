package migrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Applying 2026 schema refinement to guild_configs...")
		_, err := db.ExecContext(ctx, `
			ALTER TABLE guild_configs ALTER COLUMN deletion_status DROP DEFAULT;
			ALTER TABLE guild_configs ALTER COLUMN deletion_status TYPE varchar(20) USING deletion_status::text;
			ALTER TABLE guild_configs ALTER COLUMN deletion_status SET DEFAULT 'none';
			ALTER TABLE guild_configs ALTER COLUMN deletion_status SET NOT NULL;
			ALTER TABLE guild_configs ALTER COLUMN signup_emoji TYPE varchar(64);
			DROP TYPE IF EXISTS deletion_status_enum CASCADE;
		`)
		return err
	}, func(ctx context.Context, db *bun.DB) error {
		return nil
	})
}
