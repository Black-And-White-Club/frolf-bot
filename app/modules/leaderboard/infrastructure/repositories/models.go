package leaderboarddb

import "github.com/uptrace/bun"

type ServiceUpdateTagSource string

// Constants for ServiceUpdateTagSource.
const (
	ServiceUpdateTagSourceProcessScores ServiceUpdateTagSource = "processScores"
	ServiceUpdateTagSourceManual        ServiceUpdateTagSource = "manual"
	ServiceUpdateTagSourceCreateUser    ServiceUpdateTagSource = "createUser"
)

// Leaderboard represents a leaderboard with entries.
type Leaderboard struct {
	bun.BaseModel     `bun:"table:leaderboards,alias:l"`
	ID                int64                  `bun:"id,pk,autoincrement"`
	LeaderboardData   map[int]string         `bun:"leaderboard_data,type:jsonb,notnull"`
	IsActive          bool                   `bun:"is_active,notnull"`
	ScoreUpdateSource ServiceUpdateTagSource `bun:"score_update_source"`                 // Added to track the source of the update (e.g., from Score module, manual, etc.)
	ScoreUpdateID     string                 `bun:"score_update_id,nullzero,default:''"` // Added to store the identifier from the Score module (e.g., round ID) - make this nullable
}
