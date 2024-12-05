package models

import (
	"github.com/uptrace/bun"
)

// UpdateTagSource represents the source of a tag update.
type UpdateTagSource string

// Constants for UpdateTagSource.
const (
	UpdateTagSourceProcessScores UpdateTagSource = "processScores"
	UpdateTagSourceManual        UpdateTagSource = "manual"
	UpdateTagSourceCreateUser    UpdateTagSource = "createUser"
)

// Leaderboard represents a leaderboard with entries.
type Leaderboard struct {
	bun.BaseModel `bun:"table:leaderboards,alias:l"`

	ID              int64              `bun:"id,pk,autoincrement"`
	LeaderboardData []LeaderboardEntry `bun:"leaderboard_data,notnull,array"`
	Active          bool               `bun:"active,notnull"`
}

// LeaderboardEntry represents an entry in a leaderboard.
type LeaderboardEntry struct {
	DiscordID string `json:"discordID"`
	TagNumber int    `json:"tagNumber"`
}
