package roundevents

import (
	"time"

	roundtypes "github.com/Black-And-White-Club/tcr-bot/app/modules/round/domain/types"
	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/infrastructure/repositories"
)

// Stream names
const (
	RoundStreamName       = "round"
	UserStreamName        = "user"
	LeaderboardStreamName = "leaderboard"
	ScoreStreamName       = "score"
)

// Round-related events
const (
	RoundCreateRequest         = "round.create.request"
	RoundCreateResponse        = "round.create.response"
	RoundCreated               = "round.created"
	RoundUpdateRequest         = "round.update.request"
	RoundUpdateResponse        = "round.update.response"
	RoundUpdated               = "round.updated"
	RoundDeleteRequest         = "round.delete.request"
	RoundDeleteResponse        = "round.delete.response"
	RoundDeleted               = "round.deleted"
	ParticipantResponse        = "round.participant.response"
	ScoreUpdated               = "round.score.updated"
	RoundFinalized             = "round.finalized"
	GetUserRoleRequest         = "user.get.role.request"
	GetUserRoleResponse        = "user.get.role.response"
	RoundReminder              = "round.reminder"
	RoundStateUpdated          = "round.state.updated"
	RoundStarted               = "round.started"
	GetTagNumberRequest        = "leaderboard.get.tag.number.request"
	GetTagNumberResponse       = "leaderboard.get.tag.number.response"
	ParticipantJoined          = "round.participant.joined"
	ProcessRoundScoresRequest  = "score.process.round.scores.request"
	ProcessRoundScoresResponse = "score.process.round.scores.response"
)

// Round Events Payloads
type RoundCreateRequestPayload struct {
	Title        string                        `json:"title"`
	Location     string                        `json:"location"`
	EventType    *string                       `json:"event_type"`
	DateTime     roundtypes.DateTimeInput      `json:"date_time"`
	State        string                        `json:"round_state"`
	Participants []roundtypes.ParticipantInput `json:"participants"`
}

type RoundCreateResponsePayload struct {
	Success bool   `json:"success"`
	RoundID string `json:"round_id"`
	Error   string `json:"error,omitempty"`
}

type RoundCreatedPayload struct {
	RoundID      string                        `json:"round_id"`
	Name         string                        `json:"name"`
	StartTime    time.Time                     `json:"start_time"`
	Participants []roundtypes.ParticipantInput `json:"participants"`
	// ... other round data ...
}

type RoundUpdateRequestPayload struct {
	RoundID   string     `json:"round_id"`
	Title     *string    `json:"title,omitempty"`
	Location  *string    `json:"location,omitempty"`
	EventType *string    `json:"event_type,omitempty"`
	Date      *time.Time `json:"date,omitempty"`
	Time      *time.Time `json:"time,omitempty"`
}

type RoundUpdateResponsePayload struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

type RoundUpdatedPayload struct {
	RoundID   string     `json:"round_id"`
	Title     *string    `json:"title,omitempty"`
	Location  *string    `json:"location,omitempty"`
	EventType *string    `json:"event_type,omitempty"`
	Date      *time.Time `json:"date,omitempty"`
	Time      *time.Time `json:"time,omitempty"`
}

type RoundDeleteRequestPayload struct {
	RoundID string `json:"round_id"`
}

type RoundDeleteResponsePayload struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

type RoundDeletedPayload struct {
	RoundID string             `json:"round_id"`
	State   rounddb.RoundState `json:"state"`
}

type ParticipantResponsePayload struct {
	RoundID     string `json:"round_id"`
	Participant string `json:"participant"` // Discord ID
	Response    string `json:"response"`    // "accept", "tentative", or "decline"
}

type ScoreUpdatedPayload struct {
	RoundID     string                  `json:"round_id"`
	Participant string                  `json:"participant"` // Discord ID
	Score       int                     `json:"score"`
	UpdateType  rounddb.ScoreUpdateType `json:"update_type"`
}

type RoundFinalizedPayload struct {
	RoundID string             `json:"round_id"`
	Scores  []ParticipantScore `json:"scores"`
}

type ParticipantScore struct {
	DiscordID string `json:"discord_id"`
	TagNumber string `json:"tag_number"`
	Score     int    `json:"score"`
}

type GetUserRoleRequestPayload struct {
	DiscordID string `json:"discord_id"`
}

type GetUserRoleResponsePayload struct {
	DiscordID string `json:"discord_id"`
	Role      string `json:"role"`
	Error     string `json:"error,omitempty"`
}

type RoundReminderPayload struct {
	RoundID      string `json:"round_id"`
	ReminderType string `json:"reminder_type"` // e.g., "one_hour", "thirty_minutes"
}

type RoundStateUpdatedPayload struct {
	RoundID string             `json:"round_id"`
	State   rounddb.RoundState `json:"state"`
}

type RoundStartedPayload struct {
	RoundID      string             `json:"round_id"`
	State        rounddb.RoundState `json:"state"`
	Participants []Participant      `json:"participants"`
}

// Participant represents a participant in a round with their tag number.
type Participant struct {
	DiscordID string `json:"discord_id"`
	TagNumber int    `json:"tag_number"`
}

type GetTagNumberRequestPayload struct {
	DiscordID string `json:"discord_id"`
}

type GetTagNumberResponsePayload struct {
	DiscordID string `json:"discord_id"`
	TagNumber int    `json:"tag_number"`
	Error     string `json:"error,omitempty"`
}

type ParticipantJoinedPayload struct {
	RoundID     string `json:"round_id"`
	Participant string `json:"participant"`
	TagNumber   int    `json:"tag_number,omitempty"`
	Response    string `json:"response"`
}

// SendScoresPayload represents the event to send scores to the score module.
type SendScoresPayload struct {
	RoundID string             `json:"round_id"`
	Scores  []ParticipantScore `json:"scores"`
}
