package scoreevents

// Stream names
const (
	ScoreStreamName       = "score"
	RoundStreamName       = "round"
	UserStreamName        = "user"
	LeaderboardStreamName = "leaderboard"
)

// Event subjects.
const (
	ProcessRoundScoresRequest  = "round.process.scores.request" // Event from round module
	LeaderboardUpdateRequested = "leaderboard.update.request"   // Event to leaderboard module
	ScoreCorrectionRequest     = "score.correction.request"     // Event for manual score correction
	ScoreCorrectionError       = "score.correction.error"
	ScoreCorrectionSuccess     = "score.correction.success"
)

// --- Payloads ---

// ProcessRoundScoresRequestPayload is the payload for the ProcessRoundScoresRequest event.
type ProcessRoundScoresRequestPayload struct {
	RoundID string             `json:"round_id"`
	Scores  []ParticipantScore `json:"scores"`
}

// ParticipantScore represents a single score entry with DiscordID, TagNumber, and Score.
// Changed Score to float64
type ParticipantScore struct {
	DiscordID string  `json:"discord_id"`
	TagNumber int     `json:"tag_number"`
	Score     float64 `json:"score"`
}

// LeaderboardUpdateRequestedPayload is the payload for the LeaderboardUpdateRequested event.
type LeaderboardUpdateRequestedPayload struct {
	RoundID string             `json:"round_id"`
	Scores  []ParticipantScore `json:"scores"`
}

// ScoreUpdateRequestPayload is the payload for the ScoreCorrectionRequested event.
type ScoreUpdateRequestPayload struct {
	RoundID     string `json:"round_id"`
	Participant string `json:"participant"` // Discord ID of the participant
	Score       *int   `json:"score"`       // New score (cannot be nil in this context)
	TagNumber   int    `json:"tag_number"`
}

// ScoreUpdateErrorPayload is the payload for the ScoreCorrectionError event
type ScoreUpdateErrorPayload struct {
	CorrelationID string                     `json:"correlation_id"`
	Request       *ScoreUpdateRequestPayload `json:"score_update_request"` // Include original request
	Error         string                     `json:"error"`
}

// ScoreUpdateSuccessPayload is the payload for the ScoreCorrectionSuccess event.
type ScoreUpdateSuccessPayload struct {
	RoundID   string `json:"round_id"`
	DiscordID string `json:"discord_id"`
	NewScore  int    `json:"new_score"`
	TagNumber string `json:"tag_number"`
}
