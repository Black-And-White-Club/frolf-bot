package leaderboarddb

import "github.com/uptrace/bun"

// LeaderboardEntry represents an entry in a leaderboard.
type LeaderboardEntry struct {
	DiscordID string `json:"discordID"`
	TagNumber int    `json:"tagNumber"`
}

type ServiceUpdateTagSource string

// Constants for ServiceUpdateTagSource.
const (
	ServiceUpdateTagSourceProcessScores ServiceUpdateTagSource = "processScores"
	ServiceUpdateTagSourceManual        ServiceUpdateTagSource = "manual"
	ServiceUpdateTagSourceCreateUser    ServiceUpdateTagSource = "createUser"
)

// Leaderboard represents a leaderboard with entries.
type Leaderboard struct {
	bun.BaseModel   `bun:"table:leaderboards,alias:l"`
	ID              int64          `bun:"id,pk,autoincrement"`
	LeaderboardData map[int]string `bun:"leaderboard_data,notnull"` // Using a map for efficient access
	Active          bool           `bun:"active,notnull"`
}
