package roundmigrations

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

/*
RATIONALE

This migration introduces relational round groups while preserving
backwards compatibility with existing rounds. It also adds the
`teams` column to the rounds table to match the module Round type.
It enforces NOT NULL constraints on description and location
with safe backfill.
*/

// ---------------- SNAPSHOT STRUCTS ----------------

type roundTableV20260123 struct {
	bun.BaseModel `bun:"table:rounds,alias:r"`

	ID           uuid.UUID `bun:"id,type:uuid"`
	Mode         string    `bun:"mode"`
	Participants []byte    `bun:"participants,type:jsonb"`
	Teams        []byte    `bun:"teams,type:jsonb"` // <-- added
}

type roundGroupTableV20260123 struct {
	bun.BaseModel `bun:"table:round_groups,alias:rg"`

	ID        uuid.UUID `bun:"id,pk,type:uuid,notnull"`
	RoundID   uuid.UUID `bun:"round_id,type:uuid,notnull"`
	GroupName string    `bun:"group_name,notnull"`

	TotalScore *int      `bun:"total_score,nullzero"`
	CreatedAt  time.Time `bun:",nullzero,notnull,default:current_timestamp"`
	UpdatedAt  time.Time `bun:",nullzero,notnull,default:current_timestamp"`
}

type roundGroupMemberTableV20260123 struct {
	bun.BaseModel `bun:"table:round_group_members,alias:rgm"`

	GroupID uuid.UUID `bun:"group_id,type:uuid,notnull"`
	UserID  *string   `bun:"user_id,nullzero"`
	RawName string    `bun:"raw_name,notnull"`
}

// Legacy participant snapshot
type legacyParticipant struct {
	UserID string `json:"user_id"`
}

func init() {
	Migrations.MustRegister(
		func(ctx context.Context, db *bun.DB) error {
			fmt.Println("Applying migration: round groups + teams + enforce NOT NULL description/location")

			// ---------------------------------------------------
			// Backfill NULL description/location
			// ---------------------------------------------------
			_, err := db.ExecContext(ctx, `
				UPDATE rounds
				SET description = ''
				WHERE description IS NULL
			`)
			if err != nil {
				return fmt.Errorf("backfill description failed: %w", err)
			}

			_, err = db.ExecContext(ctx, `
				UPDATE rounds
				SET location = ''
				WHERE location IS NULL
			`)
			if err != nil {
				return fmt.Errorf("backfill location failed: %w", err)
			}

			// ---------------------------------------------------
			// Add mode column (safe default)
			// ---------------------------------------------------
			_, err = db.ExecContext(ctx, `
				ALTER TABLE IF EXISTS rounds
				ADD COLUMN IF NOT EXISTS mode VARCHAR(50) NOT NULL DEFAULT 'SINGLES'
			`)
			if err != nil {
				return fmt.Errorf("add mode column failed: %w", err)
			}

			// ---------------------------------------------------
			// Add teams column (JSONB) if missing
			// ---------------------------------------------------
			_, err = db.ExecContext(ctx, `
				ALTER TABLE IF EXISTS rounds
				ADD COLUMN IF NOT EXISTS teams JSONB NOT NULL DEFAULT '[]'::jsonb
			`)
			if err != nil {
				return fmt.Errorf("add teams column failed: %w", err)
			}

			// ---------------------------------------------------
			// Apply NOT NULL constraints
			// ---------------------------------------------------
			_, err = db.ExecContext(ctx, `
				ALTER TABLE rounds
				ALTER COLUMN description SET NOT NULL,
				ALTER COLUMN location SET NOT NULL
			`)
			if err != nil {
				return fmt.Errorf("set description/location NOT NULL failed: %w", err)
			}

			// ---------------------------------------------------
			// Create round_groups table
			// ---------------------------------------------------
			_, err = db.NewCreateTable().
				Model((*roundGroupTableV20260123)(nil)).
				ForeignKey(`
					(round_id)
					REFERENCES rounds (id)
					ON DELETE CASCADE
				`).
				IfNotExists().
				Exec(ctx)
			if err != nil {
				return err
			}

			// ---------------------------------------------------
			// Create round_group_members table
			// ---------------------------------------------------
			_, err = db.NewCreateTable().
				Model((*roundGroupMemberTableV20260123)(nil)).
				ForeignKey(`
					(group_id)
					REFERENCES round_groups (id)
					ON DELETE CASCADE
				`).
				IfNotExists().
				Exec(ctx)
			if err != nil {
				return err
			}

			// ---------------------------------------------------
			// Backfill singles rounds into round_groups
			// ---------------------------------------------------
			var rounds []roundTableV20260123
			err = db.NewSelect().
				Model(&rounds).
				Where("mode = 'SINGLES'").
				Scan(ctx)
			if err != nil {
				return err
			}

			for _, r := range rounds {
				if len(r.Participants) == 0 {
					continue
				}

				var participants []legacyParticipant
				if err := json.Unmarshal(r.Participants, &participants); err != nil {
					return err
				}

				for _, p := range participants {
					groupID := uuid.New()

					group := &roundGroupTableV20260123{
						ID:        groupID,
						RoundID:   r.ID,
						GroupName: p.UserID,
					}

					_, err := db.NewInsert().Model(group).Exec(ctx)
					if err != nil {
						return err
					}

					member := &roundGroupMemberTableV20260123{
						GroupID: groupID,
						UserID:  &p.UserID,
						RawName: p.UserID,
					}

					_, err = db.NewInsert().Model(member).Exec(ctx)
					if err != nil {
						return err
					}
				}
			}

			// ---------------------------------------------------
			// Ensure teams column is non-null for existing rounds
			// ---------------------------------------------------
			_, err = db.ExecContext(ctx, `
				UPDATE rounds
				SET teams = '[]'::jsonb
				WHERE teams IS NULL
			`)
			if err != nil {
				return fmt.Errorf("backfill teams failed: %w", err)
			}

			return nil
		},
		func(ctx context.Context, db *bun.DB) error {
			fmt.Println("Rolling back migration: round groups + teams + NOT NULL description/location")

			// Drop group tables first
			_, err := db.ExecContext(ctx, `DROP TABLE IF EXISTS round_group_members`)
			if err != nil {
				return err
			}

			_, err = db.ExecContext(ctx, `DROP TABLE IF EXISTS round_groups`)
			if err != nil {
				return err
			}

			// Drop teams column
			_, err = db.ExecContext(ctx, `ALTER TABLE IF EXISTS rounds DROP COLUMN IF EXISTS teams`)
			if err != nil {
				return fmt.Errorf("drop teams column failed: %w", err)
			}

			// Drop mode column
			_, err = db.ExecContext(ctx, `ALTER TABLE IF EXISTS rounds DROP COLUMN IF EXISTS mode`)
			if err != nil {
				return fmt.Errorf("drop mode column failed: %w", err)
			}

			// Remove NOT NULL constraints
			_, err = db.ExecContext(ctx, `
				ALTER TABLE rounds
				ALTER COLUMN description DROP NOT NULL,
				ALTER COLUMN location DROP NOT NULL
			`)
			return err
		},
	)
}
