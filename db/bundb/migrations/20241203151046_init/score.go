package models

import "github.com/uptrace/bun"

// Score represents a score with DiscordID, RoundID, Score, and TagNumber.
type Score struct {
	bun.BaseModel `bun:"table:users,alias:u"`

	DiscordID string `bun:"discord_id,notnull"`
	RoundID   string `bun:"round_id,notnull"`
	Score     int    `bun:"score,notnull"`
	TagNumber int    `bun:"tag_number"`
}
