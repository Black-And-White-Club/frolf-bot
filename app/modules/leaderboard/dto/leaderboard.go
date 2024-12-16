package leaderboarddto

// LeaderboardEntry represents an entry on the leaderboard.
type LeaderboardEntry struct {
	DiscordID string `json:"discord_id"`
	TagNumber int    `json:"tag_number"`
	// You might want to add other fields like Score, UserRank, etc.
}
