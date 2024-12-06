package models

import "github.com/uptrace/bun"

// Leaderboard represents a leaderboard with entries.
type Leaderboard struct {
	bun.BaseModel `bun:"table:leaderboards,alias:l"`

	ID              int64          `bun:"id,pk,autoincrement"`
	LeaderboardData map[int]string `bun:"leaderboard_data,notnull"` // Using a map for efficient access
	Active          bool           `bun:"active,notnull"`
}
