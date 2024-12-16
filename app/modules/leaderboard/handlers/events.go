package leaderboardhandlers

import leaderboarddto "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/dto"

// LeaderboardEntriesReceivedEvent represents an event when leaderboard entries are received.
type LeaderboardEntriesReceivedEvent struct {
	Entries []leaderboarddto.LeaderboardEntry `json:"entries"`
}

// LeaderboardTagsAssignedEvent represents an event when leaderboard tags are assigned.
type LeaderboardTagsAssignedEvent struct {
	Entries []leaderboarddto.LeaderboardEntry `json:"entries"`
}

// TagSwapEvent is published when a user wants to swap their tag.
type TagSwapEvent struct {
	DiscordID    string `json:"discord_id"`
	CurrentTag   int    `json:"current_tag"`
	RequestedTag int    `json:"requested_tag"`
}
