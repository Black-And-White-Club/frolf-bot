package leaderboardevents

// LeaderboardUpdateEvent represents an event triggered to update the leaderboard.
type LeaderboardUpdateEvent struct {
	Scores []Score `json:"scores"`
}

// Score represents a single score entry with DiscordID, TagNumber, and Score.
type Score struct {
	DiscordID string `json:"discord_id"`
	TagNumber string `json:"tag_number"`
	Score     int    `json:"score"`
}

// TagAssigned is published when a user signs up with a tag.
type TagAssigned struct {
	DiscordID string `json:"discord_id"`
	TagNumber int    `json:"tag_number"`
}

// TagSwapRequest is published when a user wants to swap their tag with another.
type TagSwapRequest struct {
	RequestorID string `json:"requestor_id"`
	TargetID    string `json:"target_id"`
}

// LeaderboardEntry represents an entry on the leaderboard.
type LeaderboardEntry struct {
	TagNumber string `json:"tag_number"`
	DiscordID string `json:"discord_id"`
}

// GetLeaderboardRequest is published when another module wants to get the entire leaderboard.
type GetLeaderboardRequest struct{}

// GetTagByDiscordIDRequest is published when another module wants to get a tag by Discord ID.
type GetTagByDiscordIDRequest struct {
	DiscordID string `json:"discord_id"`
}

// CheckTagAvailabilityRequest is published when another module wants to check if a tag is available.
type CheckTagAvailabilityRequest struct {
	TagNumber int `json:"tag_number"`
}

// GetLeaderboardResponse is published in response to a GetLeaderboardRequest.
type GetLeaderboardResponse struct {
	Leaderboard []LeaderboardEntry `json:"leaderboard"`
}

// GetTagByDiscordIDResponse is published in response to a GetTagByDiscordIDRequest.
type GetTagByDiscordIDResponse struct {
	TagNumber string `json:"tag_number"`
}

// CheckTagAvailabilityResponse represents a response to a tag availability check.
type CheckTagAvailabilityResponse struct {
	IsAvailable bool `json:"is_available"`
}

const (
	LeaderboardStream                   = "leaderboard" // Stream for leaderboard-related events
	LeaderboardUpdateEventSubject       = "leaderboard.update"
	TagAssignedSubject                  = "leaderboard.tag_assigned"
	TagSwapRequestSubject               = "leaderboard.tag_swap_request"
	GetLeaderboardRequestSubject        = "leaderboard.get_leaderboard_request"
	GetTagByDiscordIDRequestSubject     = "leaderboard.get_tag_by_discord_id_request"
	GetLeaderboardResponseSubject       = "leaderboard.get_leaderboard_response"
	GetTagByDiscordIDResponseSubject    = "leaderboard.get_tag_by_discord_id_response"
	CheckTagAvailabilityRequestSubject  = "leaderboard.check_tag_availability_request" // Subject for checking tag availability (subscribed to by leaderboard module)
	CheckTagAvailabilityResponseSubject = "leaderboard.check_tag_availability_response"
)
