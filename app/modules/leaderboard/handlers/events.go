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
	DiscordID string `json:"discord_id"`
	UserTag   int    `json:"user_tag"`   // Tag of the user requesting the swap
	TargetTag int    `json:"target_tag"` // Tag of the user they want to swap with
}

// UserTagResponseEvent represents an event containing a user's tag number.
type UserTagResponseEvent struct {
	UserID    string `json:"user_id"`
	TagNumber int    `json:"tag_number"`
}

// TagAvailabilityResponse represents a response to a tag availability check.
type TagAvailabilityResponse struct {
	TagNumber   int  `json:"tag_number"`
	IsAvailable bool `json:"is_available"`
}

// ParticipantTagResponseEvent represents a response containing a participant's tag number.
type ParticipantTagResponseEvent struct {
	ParticipantID string `json:"participant_id"`
	TagNumber     int    `json:"tag_number"`
}

// AssignTagEvent represents an event to assign a tag to a user.
type AssignTagEvent struct {
	DiscordID string `json:"discord_id"`
	TagNumber int    `json:"tag_number"`
}
