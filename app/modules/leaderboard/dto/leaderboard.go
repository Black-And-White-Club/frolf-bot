package leaderboarddto

// LeaderboardEntry represents an entry on the leaderboard.
type LeaderboardEntry struct {
	DiscordID string `json:"discord_id"`
	TagNumber int    `json:"tag_number"`
	// You might want to add other fields like Score, UserRank, etc.
}

// UpdateLeaderboardInput represents the input for updating the leaderboard.
type UpdateLeaderboardInput struct {
	Entries []LeaderboardEntry `json:"entries"`
	// Add other fields if needed, like a reason for the update
}

// ReceiveScoresInput represents the input for receiving scores.
type ReceiveScoresInput struct {
	Scores []ScoreInput `json:"scores"`
	// Add other fields if needed, like round ID or tournament ID
}

// ScoreInput represents a single score entry.
type ScoreInput struct {
	DiscordID string `json:"discord_id"`
	Score     int    `json:"score"`
	// Add other fields if needed, like game ID or date
}

// AssignTagsInput represents the input for assigning tags.
type AssignTagsInput struct {
	Tags []TagInput `json:"tags"`
	// Add other fields if needed, like season ID or reason for assignment
}

// TagInput represents a single tag assignment.
type TagInput struct {
	DiscordID string `json:"discord_id"`
	TagNumber int    `json:"tag_number"`
}

// InitiateTagSwapInput represents the input for initiating a tag swap.
type InitiateTagSwapInput struct {
	DiscordID1 string `json:"discord_id_1"`
	DiscordID2 string `json:"discord_id_2"`
	// Add other fields if needed, like a reason for the swap
}

// SwapGroupsInput represents the input for swapping groups.
type SwapGroupsInput struct {
	DiscordID1 string `json:"discord_id_1"`
	DiscordID2 string `json:"discord_id_2"`
	// Add other fields if needed, like group IDs or a reason for the swap
}
