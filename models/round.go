// models/round.go
package models

import (
	"time"
)

// Round represents a single round in the tournament.
type Round struct {
	ID           int64          `bun:"id,pk,autoincrement" json:"id"`
	Title        string         `bun:"title,notnull" json:"title"`
	Location     string         `bun:"location,notnull" json:"location"`
	EventType    *string        `bun:"event_type" json:"event_type"`
	Date         time.Time      `bun:"date,notnull" json:"date"`
	Time         string         `bun:"time,notnull" json:"time"`
	Finalized    bool           `bun:"finalized,notnull" json:"finalized"`
	CreatorID    string         `bun:"discord_id,notnull" json:"creator_id"`
	State        RoundState     `bun:"state,notnull" json:"state"`
	Participants []Participant  `bun:"type:jsonb" json:"participants"`
	Scores       map[string]int `bun:"scores,type:jsonb" json:"scores"`
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
)

// Participant represents a user participating in a round.
type Participant struct {
	DiscordID string   `json:"discord_id"` // User's Discord ID
	TagNumber *int     `json:"tag_number"` // User's tag number from the leaderboard (optional)
	Response  Response `json:"response"`   // User's response (Accept/Tentative)
}

// ScheduleRoundInput represents the input data for scheduling a new round.
type ScheduleRoundInput struct {
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
}
