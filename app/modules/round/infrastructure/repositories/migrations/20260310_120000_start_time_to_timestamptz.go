package roundmigrations

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Converting start_time from jsonb to timestamptz...")
		// Use a DO block to handle both fresh databases (where bun already creates
		// start_time as timestamptz via driver.Valuer) and production (where it's jsonb).
		_, err := db.ExecContext(ctx, `
			DO $$
			DECLARE
				col_type text;
			BEGIN
				SELECT data_type INTO col_type
				FROM information_schema.columns
				WHERE table_name = 'rounds' AND column_name = 'start_time';

				IF col_type = 'jsonb' THEN
					ALTER TABLE rounds
					  ALTER COLUMN start_time TYPE timestamptz
					  USING (start_time #>> '{}')::timestamptz;
				END IF;
				-- If already timestamptz, nothing to do.
			END $$;
		`)
		if err != nil {
			return fmt.Errorf("failed to alter start_time column: %w", err)
		}
		fmt.Println("start_time column converted successfully!")
		return nil
	}, func(ctx context.Context, db *bun.DB) error {
		fmt.Println("Rolling back start_time to jsonb...")
		_, err := db.ExecContext(ctx, `
			ALTER TABLE rounds
			  ALTER COLUMN start_time TYPE jsonb
			  USING to_jsonb(start_time::text)
		`)
		if err != nil {
			return fmt.Errorf("failed to revert start_time column: %w", err)
		}
		return nil
	})
}
