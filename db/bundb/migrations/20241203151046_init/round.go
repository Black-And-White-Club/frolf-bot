// app/models/round.go
package models

import (
	"time"

	"github.com/Black-And-White-Club/tcr-bot/api/structs"
	"github.com/uptrace/bun"
)

// Round represents a single round in the tournament.
type Round struct {
	bun.BaseModel `bun:"table:rounds,alias:r"`

	ID        int64              `bun:"id,pk,autoincrement"`
	Title     string             `bun:"title,notnull"`
	Location  string             `bun:"location,notnull"`
	EventType *string            `bun:"event_type"`
	Date      time.Time          `bun:"date,notnull"`
	Time      string             `bun:"time,notnull"`
	Finalized bool               `bun:"finalized,notnull"`
	CreatorID string             `bun:"discord_id,notnull"`
	State     structs.RoundState `bun:"state,notnull"`

	Participants structs.Participant `bun:"type:jsonb"`                 // Store participants as JSONB
	Scores       map[string]int      `bun:"scores,type:jsonb,nullzero"` // Map DiscordID to scores
}
