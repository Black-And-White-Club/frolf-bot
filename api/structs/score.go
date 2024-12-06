package structs

// Score represents a score with DiscordID, RoundID, Score, and TagNumber.
type Score struct {
	DiscordID string `json:"discord_id"`
	RoundID   string `json:"round_id"`
	Score     int    `json:"score"`
	TagNumber int    `json:"tag_number"`
}

// ScoreInput represents the input data for creating or updating a score.
type ScoreInput struct {
	DiscordID string `json:"discordID"`
	Score     int    `json:"score"`
	TagNumber *int   `json:"tagNumber"`
}
