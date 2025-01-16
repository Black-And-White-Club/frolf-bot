package scoreevents

// Stream names
const (
	ScoreStreamName       = "score"
	RoundStreamName       = "round"
	UserStreamName        = "user"
	LeaderboardStreamName = "leaderboard"
)

// Score-related events
const (
	ScoresReceivedEventSubject    = "score.received"
	ScoreCorrectedEventSubject    = "score.corrected"
	ProcessedScoresEventSubject   = "score.processed"
	LeaderboardUpdateEventSubject = "leaderboard.update"
)

// ScoresReceivedEvent represents the event when scores are received from the round module.
type ScoresReceivedEvent struct {
	RoundID string  `json:"round_id"`
	Scores  []Score `json:"scores"`
}

// Score represents a single score entry with DiscordID, TagNumber, and Score.
type Score struct {
	DiscordID string `json:"discord_id"`
	TagNumber string `json:"tag_number"`
	Score     int    `json:"score"`
}

// LeaderboardUpdateEvent represents an event triggered to update the leaderboard.
type LeaderboardUpdateEvent struct {
	RoundID string  `json:"round_id"`
	Scores  []Score `json:"scores"`
}

// ScoreCorrectedEvent represents an event for a corrected score.
type ScoreCorrectedEvent struct {
	RoundID   string `json:"round_id"`
	DiscordID string `json:"discord_id"`
	NewScore  int    `json:"new_score"`
	TagNumber string `json:"tag_number"`
	Error     string `json:"error,omitempty"` // Add Error field
	Success   bool   `json:"success"`         // Add Success field
}

// ProcessedScoresEvent represents an event indicating that scores have been processed.
type ProcessedScoresEvent struct {
	RoundID string  `json:"round_id"`
	Scores  []Score `json:"scores"`
	Error   string  `json:"error,omitempty"` // Add Error field
	Success bool    `json:"success"`         // Add Success field
}
