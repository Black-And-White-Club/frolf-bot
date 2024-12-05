package models

import "github.com/uptrace/bun"

// ScoreInput represents the input data for creating or updating a score.
type ScoreInput struct {
	DiscordID string `json:"discordID"`
	Score     int    `json:"score"`
	TagNumber *int   `json:"tagNumber"`
}

// Score represents a score with DiscordID, RoundID, Score, and TagNumber.
type Score struct {
	bun.BaseModel `bun:"table:scores,alias:s"`

	DiscordID string `bun:"discord_id,notnull"`
	RoundID   string `bun:"round_id,notnull"`
	Score     int    `bun:"score,notnull"`
	TagNumber int    `bun:"tag_number"`
}
