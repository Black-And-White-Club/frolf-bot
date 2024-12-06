package structs

// Leaderboard represents a leaderboard with entries.
type Leaderboard struct {
	ID              int64          `json:"id"`
	LeaderboardData map[int]string `json:"leaderboard_data"`
	Active          bool           `json:"active"`
}

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
