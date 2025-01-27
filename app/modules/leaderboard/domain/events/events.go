package leaderboardevents

import (
	leaderboardtypes "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/domain/types"
)

// Stream names
const (
	LeaderboardStreamName = "leaderboard"
	UserStreamName        = "user"
	RoundStreamName       = "round"
	ScoreStreamName       = "score"
)

// Leaderboard-related events
const (
	// Leaderboard Update
	RoundFinalized             = "leaderboard.round.finalized"  // From Score module
	LeaderboardUpdateRequested = "leaderboard.update.requested" // Internal to Leaderboard module
	LeaderboardUpdated         = "leaderboard.updated"          // Internal or external
	LeaderboardUpdateFailed    = "leaderboard.update.failed"    // Internal or external
	DeactivateOldLeaderboard   = "leaderboard.deactivate"       // Internal

	// Tag Assignment
	TagAvailabilityCheckRequest       = "leaderboard.tag.availability.check.request" // From User module
	LeaderboardTagAssignmentRequested = "leaderboard.tag.assignment.requested"       // Internal to Leaderboard module
	LeaderboardTagAssignmentFailed    = "leaderboard.tag.assignment.failed"          // Internal to Leaderboard module
	TagAssigned                       = "leaderboard.tag.assigned"                   // Internal to Leaderboard module
	TagAvailable                      = "leaderboard.tag.available"                  // Internal to Leaderboard module
	TagUnavailable                    = "user.tag.unavailable"

	// Tag Swap
	TagSwapRequested = "leaderboard.tag.swap.requested" // External
	TagSwapInitiated = "leaderboard.tag.swap.initiated" // Internal
	TagSwapFailed    = "leaderboard.tag.swap.failed"    // Internal
	TagSwapProcessed = "leaderboard.tag.swap.processed" // Internal

	// Leaderboard Requests
	GetLeaderboardRequest  = "leaderboard.get.request"
	GetLeaderboardResponse = "leaderboard.get.response"

	// Tag Requests
	GetTagByDiscordIDRequest  = "leaderboard.get.tag.by.discord.id.request"
	GetTagByDiscordIDResponse = "leaderboard.get.tag.by.discord.id.response"
)

// -- Event Payloads --

// RoundFinalizedPayload is the payload for the RoundFinalized event.
type RoundFinalizedPayload struct {
	RoundID               string   `json:"round_id"`
	SortedParticipantTags TagOrder `json:"sorted_participant_tags"` // Slice of "tag:discordID" strings
}

// TagAssignedPayload is the payload for the TagAssigned event.
type TagAssignedPayload struct {
	DiscordID    leaderboardtypes.DiscordID `json:"discord_id"`
	TagNumber    int                        `json:"tag_number"`
	AssignmentID string                     `json:"assignment_id"`
}

// TagAvailablePayload is the payload for the TagAvailable event.
type TagAvailablePayload struct {
	DiscordID    leaderboardtypes.DiscordID `json:"discord_id"`
	TagNumber    int                        `json:"tag_number"`
	AssignmentID string                     `json:"assignment_id"`
}

// TagUnavailablePayload is the payload for the TagUnavailable event.
type TagUnavailablePayload struct {
	DiscordID leaderboardtypes.DiscordID `json:"discord_id"`
	TagNumber int                        `json:"tag_number"`
	Reason    string                     `json:"reason"`
}

// TagAssignmentRequestedPayload is the payload for the TagAssignmentRequested event.
type TagAssignmentRequestedPayload struct {
	DiscordID  leaderboardtypes.DiscordID `json:"discord_id"`
	TagNumber  int                        `json:"tag_number"`
	UpdateID   string                     `json:"update_id"`
	Source     string                     `json:"source"`
	UpdateType string                     `json:"update_type"`
}

// LeaderboardUpdateRequestedPayload is the payload for the LeaderboardUpdateRequested event.
type LeaderboardUpdateRequestedPayload struct {
	RoundID               string   `json:"round_id"`
	SortedParticipantTags []string `json:"sorted_participant_tags"`
	Source                string   `json:"source"`    // "round", "manual"
	UpdateID              string   `json:"update_id"` // round ID or manual update identifier
}

// LeaderboardUpdatedPayload is the payload for the LeaderboardUpdated event.
type LeaderboardUpdatedPayload struct {
	LeaderboardID int64  `json:"leaderboard_id"`
	RoundID       string `json:"round_id"`
}

// LeaderboardUpdateFailedPayload is the payload for the LeaderboardUpdateFailed event.
type LeaderboardUpdateFailedPayload struct {
	RoundID string `json:"round_id"`
	Reason  string `json:"reason"` // Reason for the failure
}

// DeactivateOldLeaderboardPayload is the payload for the DeactivateOldLeaderboard event.
type DeactivateOldLeaderboardPayload struct {
	LeaderboardID int64 `json:"leaderboard_id"`
}

// TagAssignmentFailedPayload is the payload for the TagAssignmentFailed event.
type TagAssignmentFailedPayload struct {
	DiscordID  leaderboardtypes.DiscordID `json:"discord_id"`
	TagNumber  int                        `json:"tag_number"`
	UpdateID   string                     `json:"update_id"`
	Source     string                     `json:"source"`
	UpdateType string                     `json:"update_type"`
	Reason     string                     `json:"reason"`
}

// TagSwapRequestedPayload is the payload for the TagSwapRequested event.
type TagSwapRequestedPayload struct {
	RequestorID string `json:"requestor_id"`
	TargetID    string `json:"target_id"`
}

// TagSwapInitiatedPayload is the payload for the TagSwapInitiated event.
type TagSwapInitiatedPayload struct {
	RequestorID string `json:"requestor_id"`
	TargetID    string `json:"target_id"`
}

// TagSwapFailedPayload is the payload for the TagSwapFailed event.
type TagSwapFailedPayload struct {
	RequestorID string `json:"requestor_id"`
	TargetID    string `json:"target_id"`
	Reason      string `json:"reason"`
}

// TagSwapProcessedPayload is the payload for the TagSwapProcessed event.
type TagSwapProcessedPayload struct {
	RequestorID string `json:"requestor_id"`
	TargetID    string `json:"target_id"`
}

// GetLeaderboardRequestPayload is the payload for the GetLeaderboardRequest event.
type GetLeaderboardRequestPayload struct{} // Empty, as no data is needed for this request

// LeaderboardEntry represents an entry on the leaderboard.
type LeaderboardEntry struct {
	TagNumber string                     `json:"tag_number"`
	DiscordID leaderboardtypes.DiscordID `json:"discord_id"`
}

// GetLeaderboardResponsePayload is the payload for the GetLeaderboardResponse event.
type GetLeaderboardResponsePayload struct {
	Leaderboard []LeaderboardEntry `json:"leaderboard"`
}

// GetTagByDiscordIDRequestPayload is the payload for the GetTagByDiscordIDRequest event.
type GetTagByDiscordIDRequestPayload struct {
	DiscordID leaderboardtypes.DiscordID `json:"discord_id"`
}

// GetTagByDiscordIDResponsePayload is the payload for the GetTagByDiscordIDResponse event.
type GetTagByDiscordIDResponsePayload struct {
	TagNumber int `json:"tag_number"`
}

// TagAvailabilityCheckRequestedPayload is the payload for the TagAvailabilityCheckRequested event.
type TagAvailabilityCheckRequestedPayload struct {
	TagNumber int                        `json:"tag_number"`
	DiscordID leaderboardtypes.DiscordID `json:"discord_id"`
}

// -- Helper Types --

// TagOrder represents the order of tags.
type TagOrder []string
