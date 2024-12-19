package roundevents

import (
	"time"

	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
)

// RoundCreatedEvent is published when a new round is created.
type RoundCreatedEvent struct {
	RoundID      string    `json:"round_id"` // Using string for consistency with other IDs
	Name         string    `json:"name"`
	StartTime    time.Time `json:"start_time"`
	Participants []string  `json:"participants"` // Discord IDs of participants
	// ... other round data from rounddto.CreateRoundInput
}

// RoundUpdatedEvent is published when a round is updated.
type RoundUpdatedEvent struct {
	RoundID   string     `json:"round_id"` // Add RoundID field
	Title     *string    `json:"title,omitempty"`
	Location  *string    `json:"location,omitempty"`
	EventType *string    `json:"event_type,omitempty"`
	Date      *time.Time `json:"date,omitempty"`
	Time      *time.Time `json:"time,omitempty"`
}

// RoundDeletedEvent is published when a round is deleted.
type RoundDeletedEvent struct {
	RoundID string             `json:"round_id"`
	State   rounddb.RoundState `json:"state"` // Add the State field
}

// ParticipantResponseEvent is published when a participant responds to a round invitation.
type ParticipantResponseEvent struct {
	RoundID     string `json:"round_id"`
	Participant string `json:"participant"` // Discord ID
	Response    string `json:"response"`    // "accept", "tentative", or "decline"
}

// ScoreUpdatedEvent is published when a participant updates their score.
type ScoreUpdatedEvent struct {
	RoundID     string                  `json:"round_id"`
	Participant string                  `json:"participant"` // Discord ID
	Score       int                     `json:"score"`
	UpdateType  rounddb.ScoreUpdateType `json:"update_type"`
}

// RoundFinalizedEvent is published when a round is finalized.
type RoundFinalizedEvent struct {
	RoundID string             `json:"round_id"`
	Scores  []ParticipantScore `json:"scores"`
}

type ParticipantScore struct {
	DiscordID string `json:"discord_id"`
	TagNumber string `json:"tag_number"` // Using string for consistency, even if it's a number
	Score     int    `json:"score"`
}

// GetUserRoleRequestEvent is published to request the role of a user.
type GetUserRoleRequestEvent struct {
	DiscordID string `json:"discord_id"`
}

// GetUserRoleResponseEvent is published in response to GetUserRoleRequestEvent.
type GetUserRoleResponseEvent struct {
	DiscordID string `json:"discord_id"`
	Role      string `json:"role"`
}

// RoundReminderEvent is published to trigger a round reminder.
type RoundReminderEvent struct {
	RoundID      string `json:"round_id"`
	ReminderType string `json:"reminder_type"` // e.g., "one_hour", "thirty_minutes"
}

// RoundStateUpdatedEvent is published when the state of a round changes.
type RoundStateUpdatedEvent struct {
	RoundID string             `json:"round_id"`
	State   rounddb.RoundState `json:"state"`
}

// RoundCreatedFailedEvent is published when a round creation fails.
type RoundCreatedFailedEvent struct {
	Reason string `json:"reason"`
}

// GetTagNumberRequest is published to request the tag number of a user.
type GetTagNumberRequest struct {
	DiscordID string `json:"discord_id"` // Use DiscordID to request the tag number
}

// GetTagNumberResponseEvent is published in response to GetTagNumberRequest.
type GetTagNumberResponseEvent struct {
	DiscordID string `json:"discord_id"`
	TagNumber int    `json:"tag_number"` // Use int for the tag number
}

// ParticipantJoinedRoundEvent is published when a participant joins a round.
type ParticipantJoinedRoundEvent struct {
	RoundID     string `json:"round_id"`
	Participant string `json:"participant"`          // Discord ID
	TagNumber   int    `json:"tag_number,omitempty"` // Include tag number if available
	Response    string `json:"response"`             // "accept", "tentative", or "decline"
}

const (
	GetTagNumberRequestSubject  = "round.get_tag_number_request"
	GetTagNumberResponseSubject = "round.get_tag_number_response"
	LeaderboardStream           = "leaderboard"
	GetUserRoleRequestSubject   = "user.get_role_request"
	GetUserRoleResponseSubject  = "user.get_role_response"
	UserStream                  = "user"
	RoundStream                 = "round"
	ParticipantResponseSubject  = "round.participant.response"
	ScoreUpdatedSubject         = "round.score.updated"
	RoundFinalizedSubject       = "round.finalized_event"
	RoundCreatedSubject         = "round.created"
	RoundUpdatedSubject         = "round.updated"
	RoundDeletedSubject         = "round.deleted"
)

// RoundStartedEvent is published when a round starts.
type RoundStartedEvent struct {
	RoundID      string             `json:"round_id"`
	State        rounddb.RoundState `json:"state"`
	Participants []Participant      `json:"participants"`
}

// Participant represents a participant in a round with their tag number.
type Participant struct {
	DiscordID string `json:"discord_id"`
	TagNumber int    `json:"tag_number"`
}

// SendScoresEvent represents the event to send scores to the score module.
type SendScoresEvent struct {
	RoundID string             `json:"round_id"`
	Scores  []ParticipantScore `json:"scores"`
}
