package scoredto

// ScoreDTO represents individual scores in the ScoresProcessedEvent.
type Score struct {
	DiscordID string `json:"discord_id"`
	Score     int    `json:"score"`
	TagNumber int    `json:"tag_number"`
}
