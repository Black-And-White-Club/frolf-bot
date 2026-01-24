package roundmigrations

import (
	"context"
	"fmt"
	"time"

	"github.com/uptrace/bun"
)

func init() {
    Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
        fmt.Println("Backfilling rounds.teams NULL -> []...")

        // Use a short-lived context so this migration cannot hang indefinitely.
        ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
        defer cancel()

        // If the rounds table doesn't exist yet (migration ordering), skip this
        // backfill rather than failing the entire migration run.
        var exists bool
        if err := db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'rounds')`).Scan(&exists); err != nil {
            return fmt.Errorf("failed to check rounds table existence: %w", err)
        }
        if !exists {
            fmt.Println("rounds table does not exist, skipping backfill")
            return nil
        }

        res, err := db.ExecContext(ctx, `UPDATE rounds SET teams = '[]'::jsonb WHERE teams IS NULL;`)
        if err != nil {
            return fmt.Errorf("failed to update rounds.teams: %w", err)
        }

        if ra, err := res.RowsAffected(); err == nil {
            fmt.Printf("rows affected: %d\n", ra)
        }

        return nil
    }, func(ctx context.Context, db *bun.DB) error {
        // Non-destructive rollback: restoring NULL values is not safe without a
        // prior backup. Documented rollback should use a DB snapshot/restore.
        fmt.Println("Rollback (no-op): restoring previous NULLs requires a DB backup/restore")
        return nil
    })
}
