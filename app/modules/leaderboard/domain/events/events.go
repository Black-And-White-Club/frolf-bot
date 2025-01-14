package leaderboardevents

import "fmt"

// Stream names
const (
	LeaderboardStreamName = "leaderboard"
	UserStreamName        = "user"
)

// Leaderboard-related events
const (
	LeaderboardUpdatedSubject           = "leaderboard.updated"
	TagAssignedSubject                  = "leaderboard.tag.assigned"
	TagSwapRequestedSubject             = "leaderboard.tag.swap.requested"
	GetLeaderboardRequestSubject        = "leaderboard.get.leaderboard.request"
	GetTagByDiscordIDRequestSubject     = "leaderboard.get.tag.by.discord.id.request"
	CheckTagAvailabilityRequestSubject  = "leaderboard.check.tag.availability.request"
	GetLeaderboardResponseSubject       = "leaderboard.get.leaderboard.response"
	GetTagByDiscordIDResponseSubject    = "leaderboard.get.tag.by.discord.id.response"
	CheckTagAvailabilityResponseSubject = "leaderboard.check.tag.availability.response"
)

// LeaderboardUpdateEvent is published when the leaderboard needs to be updated.
type LeaderboardUpdateEvent struct {
	Scores []Score `json:"scores"`
}

// Score represents a single score entry with DiscordID, TagNumber, and Score.
type Score struct {
	DiscordID string `json:"discord_id"`
	TagNumber string `json:"tag_number"`
	Score     int    `json:"score"`
}

// TagAssignedEvent is published when a user signs up with a tag.
type TagAssignedEvent struct {
	DiscordID string `json:"discord_id"`
	TagNumber int    `json:"tag_number"`
}

// TagSwapRequestEvent is published when a user wants to swap their tag with another.
type TagSwapRequestEvent struct {
	RequestorID string `json:"requestor_id"`
	TargetID    string `json:"target_id"`
}

// LeaderboardEntry represents an entry on the leaderboard.
type LeaderboardEntry struct {
	TagNumber string `json:"tag_number"`
	DiscordID string `json:"discord_id"`
}

// GetLeaderboardRequestEvent is published when another module wants to get the entire leaderboard.
type GetLeaderboardRequestEvent struct{}

// GetTagByDiscordIDRequestEvent is published when another module wants to get a tag by Discord ID.
type GetTagByDiscordIDRequestEvent struct {
	DiscordID string `json:"discord_id"`
}

// CheckTagAvailabilityRequestEvent is published when another module wants to check if a tag is available.
type CheckTagAvailabilityRequestEvent struct {
	TagNumber int `json:"tag_number"`
}

// GetLeaderboardResponseEvent is published in response to a GetLeaderboardRequestEvent.
type GetLeaderboardResponseEvent struct {
	Leaderboard []LeaderboardEntry `json:"leaderboard"`
}

// GetTagByDiscordIDResponseEvent is published in response to a GetTagByDiscordIDRequestEvent.
type GetTagByDiscordIDResponseEvent struct {
	TagNumber string `json:"tag_number"`
}

// CheckTagAvailabilityResponseEvent represents a response to a tag availability check.
type CheckTagAvailabilityResponseEvent struct {
	IsAvailable bool `json:"is_available"`
}

// --- Errors ---

type ErrTagAlreadyAssigned struct {
	TagNumber int
}

func (e *ErrTagAlreadyAssigned) Error() string {
	return fmt.Sprintf("tag number %d is already assigned", e.TagNumber)
}

type ErrLeaderboardUpdateFailed struct {
	Reason string
}

func (e *ErrLeaderboardUpdateFailed) Error() string {
	return fmt.Sprintf("leaderboard update failed: %s", e.Reason)
}
