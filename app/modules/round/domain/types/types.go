// types.go
package roundtypes

import "time"

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

// IsUpcoming checks if the round is in the upcoming state.
func (r *Round) IsUpcoming() bool {
	return r.State == RoundStateUpcoming
}

// IsInProgress checks if the round is in the in-progress state.
func (r *Round) IsInProgress() bool {
	return r.State == RoundStateInProgress
}

// IsFinalized checks if the round is in the finalized state.
func (r *Round) IsFinalized() bool {
	return r.State == RoundStateFinalized
}

// AddParticipant adds a participant to the round.
func (r *Round) AddParticipant(participant Participant) {
	r.Participants = append(r.Participants, participant)
}

// ... other methods for Round if needed ...

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

// DateTimeInput represents the date and time input for creating a round.
type DateTimeInput struct {
	Date string `json:"date"`
	Time string `json:"time"`
}

// CreateRoundInput represents the input for creating a new round.
type CreateRoundInput struct {
	Title        string             `json:"title"`
	Location     string             `json:"location"`
	EventType    *string            `json:"event_type"`
	DateTime     DateTimeInput      `json:"date_time"`
	State        string             `json:"round_state"`
	Participants []ParticipantInput `json:"participants"`
}

// ParticipantInput represents the input for a participant in a round.
type ParticipantInput struct {
	DiscordID string `json:"discord_id"`
	TagNumber *int   `json:"tag_number"`
	Response  string `json:"response"`
}

// EditRoundInput represents the input data for editing a round.
type EditRoundInput struct {
	RoundID   int64     `json:"round_id"`
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
	TagNumber *int     `json:"tagNumber"`
}
