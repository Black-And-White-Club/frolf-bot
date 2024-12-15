package rounddto

import (
	"time"
)

// Round represents a single round in the tournament.
type Round struct {
	ID           int64          `json:"id"`
	Title        string         `json:"title"`
	Location     string         `json:"location"`
	EventType    *string        `json:"event_type"`
	Date         time.Time      `json:"date"`
	Time         string         `json:"time"`
	Finalized    bool           `json:"finalized"`
	CreatorID    string         `json:"creator_id"`
	State        RoundState     `json:"state"`
	Participants []Participant  `json:"participants"`
	Scores       map[string]int `json:"scores"`
}

// Response represents the possible responses for a participant.
type Response string

// Define the possible response values as constants.
const (
	ResponseAccept    Response = "ACCEPT"
	ResponseTentative Response = "TENTATIVE"
	ResponseDecline   Response = "DECLINE"
)

// RoundState represents the state of a round.
type RoundState string

// Enum constants for RoundState
const (
	RoundStateUpcoming   RoundState = "UPCOMING"
	RoundStateInProgress RoundState = "IN_PROGRESS"
	RoundStateFinalized  RoundState = "FINALIZED"
	RoundStateDleted     RoundState = "DELETED"
)

// Participant represents a user participating in a round.
type Participant struct {
	DiscordID string   `json:"discord_id"`
	TagNumber *int     `json:"tag_number"`
	Response  Response `json:"response"`
}

// ParticipantScore represents the score of a participant in a finalized round.
type ParticipantScore struct {
	DiscordID string `json:"discord_id"`
	TagNumber int    `json:"tag_number"`
	Score     int    `json:"score"`
}

// CreateRoundInput represents the input data for scheduling a new round.
type CreateRoundInput struct {
	Title     string    `json:"title"`
	Location  string    `json:"location"`
	EventType *string   `json:"eventType"`
	Date      time.Time `json:"date"`
	Time      string    `json:"time"`
	DiscordID string    `json:"discordID"`
}

// JoinRoundInput represents the input data for a participant joining a round.
type JoinRoundInput struct {
	RoundID   int64    `json:"roundID"`
	DiscordID string   `json:"discordID"`
	Response  Response `json:"response"`
}

// SubmitScoreInput represents the input data for submitting a score.
type SubmitScoreInput struct {
	RoundID   int64  `json:"roundID"`
	DiscordID string `json:"discordID"`
	Score     int    `json:"score"`
}

// EditRoundInput represents the input data for editing a round.
type EditRoundInput struct {
	Title     string    `json:"title"`
	Location  string    `json:"location"`
	EventType *string   `json:"eventType"`
	Date      time.Time `json:"date"`
	Time      string    `json:"time"`
}

// UpdateParticipantResponseInput represents the input data for updating a participant's response.
type UpdateParticipantResponseInput struct {
	RoundID   int64    `json:"roundID"`
	DiscordID string   `json:"discordID"`
	Response  Response `json:"response"`
	TagNumber *int     `json:"tagNumber"` // Add the TagNumber field
}
