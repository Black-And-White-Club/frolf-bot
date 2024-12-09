package migrations

import (
	"context"

	"github.com/Black-And-White-Club/tcr-bot/models"
	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		// SQL for creating the 'rounds' table
		_, err := db.NewCreateTable().
			Model((*models.Round)(nil)).
			Column("id", "title", "location", "event_type", "date", "time", "finalized", "discord_id", "state").
			Exec(ctx)
		if err != nil {
			return err
		}

		// Add Participants and Scores columns with JSONB type
		_, err = db.NewAddColumn().
			Model((*models.Round)(nil)).
			Table("rounds").
			Column("participants", "type:jsonb").
			Column("scores", "scores,type:jsonb").
			Exec(ctx)
		return err
	}, func(ctx context.Context, db *bun.DB) error {
		// SQL for dropping the 'rounds' table
		_, err := db.NewDropTable().
			Model((*models.Round)(nil)).
			Exec(ctx)
		return err
	})
}
