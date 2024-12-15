// app/modules/round/db/models.go
package rounddb

import (
	"time"
)

// Round represents a single round in the tournament.
type Round struct {
	ID            int64          `bun:"id,pk,autoincrement" json:"id"`
	Title         string         `bun:"title,notnull" json:"title"`
	Location      string         `bun:"location" json:"location"`
	EventType     *string        `bun:"event_type" json:"event_type"`
	Date          time.Time      `bun:"date,notnull" json:"date"`
	Time          string         `bun:"time,notnull" json:"time"`
	Finalized     bool           `bun:"finalized,notnull" json:"finalized"`
	CreatorID     string         `bun:"discord_id,notnull" json:"discord_id"`
	State         RoundState     `bun:"state,notnull" json:"state"`
	Participants  []Participant  `bun:"type:jsonb" json:"participants"`
	Scores        map[string]int `bun:"scores,type:jsonb" json:"scores"`
	PendingScores []Score        `bun:"pending_scores,type:jsonb" json:"pending_scores"` // Add this field
}

// Score represents a score record for a participant in a round.
type Score struct {
	ParticipantID string `json:"discord_id"`
	Score         int    `json:"score"`
	// ... other fields you might need for a score ...
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
	RoundStateDeleted    RoundState = "DELETED"
)

// Participant represents a user participating in a round.
type Participant struct {
	DiscordID string   `json:"discord_id"`
	TagNumber *int     `json:"tag_number"`
	Response  Response `json:"response"`
}
