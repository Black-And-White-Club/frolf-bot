package roundevents

import (
	"time"

	roundtypes "github.com/Black-And-White-Club/tcr-bot/app/modules/round/domain/types"
)

// Stream names
const (
	RoundStreamName       = "round"
	UserStreamName        = "user"
	LeaderboardStreamName = "leaderboard"
	ScoreStreamName       = "score" // Consider moving this to scoreevents
)

// Round-related events
const (
	// --- Create Round ---
	RoundCreateRequest           = "round.create.request"
	RoundValidated               = "round.validated"
	RoundDateTimeParsed          = "round.datetime.parsed"
	RoundEntityCreated           = "round.entity.created"
	RoundStored                  = "round.stored"
	RoundScheduled               = "round.scheduled"
	RoundCreated                 = "round.created"
	RoundError                   = "round.error"
	RoundUpdateError             = "round.update.error"
	RoundFinalizationError       = "round.finalization.error"
	ScoreModuleNotificationError = "score.module.notification.error"

	// --- Update Round ---
	RoundUpdateRequest   = "round.update.request"
	RoundUpdateValidated = "round.update.validated"
	RoundFetched         = "round.fetched"
	RoundEntityUpdated   = "round.entity.updated"
	RoundUpdated         = "round.updated"
	RoundUpdateSuccess   = "round.update.success"

	// --- Delete Round ---
	RoundDeleteRequest      = "round.delete.request"
	RoundDeleteValidated    = "round.delete.validated"
	RoundToDeleteFetched    = "round.to.delete.fetched"
	RoundDeleteAuthorized   = "round.delete.authorized"
	RoundDeleteUnauthorized = "round.delete.unauthorized" // NEW
	RoundDeleted            = "round.deleted"
	RoundDeleteError        = "round.delete.error"

	// --- Join Round ---
	RoundParticipantJoinRequest   = "round.participant.join.request"
	RoundParticipantJoinValidated = "round.participant.join.validated"
	RoundParticipantJoinError     = "round.participant.join.error"
	ParticipantJoined             = "round.participant.joined"

	// --- Score Round ---
	RoundScoreUpdateRequest      = "round.score.update.request"
	RoundScoreUpdateValidated    = "round.score.update.validated"
	RoundParticipantScoreUpdated = "round.participant.score.updated"
	RoundAllScoresSubmitted      = "round.all.scores.submitted"
	RoundNotAllScoresSubmitted   = "round.not.all.scores.submitted" // You might not need this one
	RoundScoreUpdateError        = "round.score.update.error"

	// --- Finalize Round ---
	RoundFinalized          = "round.finalized"
	RoundScoresNotification = "round.scores.notification"

	// --- Round Reminders ---
	RoundReminder = "round.reminder"

	// --- Round State ---
	RoundStateUpdated = "round.state.updated"

	// --- Start Round ---
	RoundStarted = "round.started"

	// --- Tag Retrieval ---
	RoundTagNumberRequest  = "round.tag.number.request"
	RoundTagNumberFound    = "round.tag.number.found"
	RoundTagNumberNotFound = "round.tag.number.notfound"

	// --- Communication with Other Modules ---
	// Leaderboard
	LeaderboardGetTagNumberRequest  = "leaderboard.get.tag.number.request"  // To be handled by the Leaderboard module
	LeaderboardGetTagNumberResponse = "leaderboard.get.tag.number.response" // To be handled by the Leaderboard module
	// Score Module
	ProcessRoundScoresRequest = "score.process.scores.request" // To be handled by the Score module from Round module

	// --- User Authorization ---
	RoundUserRoleCheckRequest = "round.user.role.check.request" // NEW
	RoundUserRoleCheckResult  = "round.user.role.check.result"  // NEW
	RoundUserRoleCheckError   = "round.user.role.check.error"   // NEW
)

// Round Events Payloads

// --- Create Round ---
type RoundCreateRequestPayload struct {
	Title        string                        `json:"title"`
	Location     string                        `json:"location"`
	EventType    *string                       `json:"event_type"`
	DateTime     roundtypes.RoundTimeInput     `json:"date_time"`
	Participants []roundtypes.ParticipantInput `json:"participants"`
}

type RoundValidatedPayload struct {
	RoundCreateRequestPayload RoundCreateRequestPayload `json:"round_create_request_payload"`
}

type RoundDateTimeParsedPayload struct {
	RoundCreateRequestPayload RoundCreateRequestPayload `json:"round_create_request_payload"`
	StartTime                 time.Time                 `json:"start_time"`
}

type RoundEntityCreatedPayload struct {
	Round roundtypes.Round `json:"round"`
}

type RoundStoredPayload struct {
	Round roundtypes.Round `json:"round"`
}

type RoundScheduledPayload struct {
	RoundID string `json:"round_id"`
}

type RoundCreatedPayload struct {
	RoundID   string    `json:"round_id"`
	Name      string    `json:"name"`
	StartTime time.Time `json:"start_time"`
}

type RoundErrorPayload struct {
	CorrelationID string                     `json:"correlation_id"`
	Round         *RoundCreateRequestPayload `json:"round"`
	Error         string                     `json:"error"`
}

// --- Update Round ---
type RoundUpdateRequestPayload struct {
	RoundID   string     `json:"round_id" validate:"required"`
	Title     *string    `json:"title,omitempty"`
	Location  *string    `json:"location,omitempty"`
	EventType *string    `json:"event_type,omitempty"`
	Date      *time.Time `json:"date,omitempty"`
	Time      *time.Time `json:"time,omitempty"`
}

type RoundUpdateValidatedPayload struct {
	RoundUpdateRequestPayload RoundUpdateRequestPayload `json:"round_update_request_payload"`
}

type RoundFetchedPayload struct {
	Round                     roundtypes.Round          `json:"round"`
	RoundUpdateRequestPayload RoundUpdateRequestPayload `json:"round_update_request_payload"`
}

type RoundEntityUpdatedPayload struct {
	Round roundtypes.Round `json:"round"`
}

type RoundUpdatedPayload struct {
	RoundID string `json:"round_id"`
}

type RoundUpdateSuccessPayload struct {
	RoundID string `json:"round_id"`
}

type RoundUpdateErrorPayload struct {
	CorrelationID      string                     `json:"correlation_id"`
	RoundUpdateRequest *RoundUpdateRequestPayload `json:"round_update_request"`
	Error              string                     `json:"error"`
}

// --- Delete Round ---
type RoundDeleteRequestPayload struct {
	RoundID                 string `json:"round_id" validate:"required"`
	RequestingUserDiscordID string `json:"requesting_user_discord_id" validate:"required"`
}

type RoundDeleteValidatedPayload struct {
	RoundDeleteRequestPayload RoundDeleteRequestPayload `json:"round_delete_request_payload"`
}

type RoundToDeleteFetchedPayload struct {
	Round                     roundtypes.Round          `json:"round"`
	RoundDeleteRequestPayload RoundDeleteRequestPayload `json:"round_delete_request_payload"`
}

type RoundDeleteAuthorizedPayload struct {
	RoundID string `json:"round_id"`
}

type RoundDeletedPayload struct {
	RoundID string `json:"round_id"`
}

type RoundDeleteErrorPayload struct {
	CorrelationID      string                     `json:"correlation_id"`
	RoundDeleteRequest *RoundDeleteRequestPayload `json:"round_delete_request"`
	Error              string                     `json:"error"`
}

// --- Join Round ---
type ParticipantJoinRequestPayload struct {
	RoundID     string `json:"round_id"`
	Participant string `json:"participant"` // Discord ID
}

type ParticipantJoinValidatedPayload struct {
	ParticipantJoinRequestPayload ParticipantJoinRequestPayload `json:"participant_join_request_payload"`
}

type RoundParticipantJoinErrorPayload struct {
	CorrelationID          string                         `json:"correlation_id"`
	ParticipantJoinRequest *ParticipantJoinRequestPayload `json:"participant_join_request"`
	Error                  string                         `json:"error"`
}

type ParticipantJoinedPayload struct {
	RoundID     string `json:"round_id"`
	Participant string `json:"participant"`
	TagNumber   int    `json:"tag_number,omitempty"`
	Response    string `json:"response"`
}

// --- Score Round ---
type ScoreUpdateRequestPayload struct {
	RoundID     string `json:"round_id"`
	Participant string `json:"participant"` // Discord ID
	Score       *int   `json:"score"`
}

type ScoreUpdateValidatedPayload struct {
	ScoreUpdateRequestPayload ScoreUpdateRequestPayload `json:"score_update_request_payload"`
}

type ParticipantScoreUpdatedPayload struct {
	RoundID     string `json:"round_id"`
	Participant string `json:"participant"` // Discord ID
	Score       int    `json:"score"`
}

type AllScoresSubmittedPayload struct {
	RoundID string `json:"round_id"`
}

type RoundScoreUpdateErrorPayload struct {
	CorrelationID      string                     `json:"correlation_id"`
	ScoreUpdateRequest *ScoreUpdateRequestPayload `json:"score_update_request"`
	Error              string                     `json:"error"`
}

// --- Finalize Round ---
type RoundFinalizedPayload struct {
	RoundID string `json:"round_id"`
}

type RoundFinalizationErrorPayload struct {
	CorrelationID string `json:"correlation_id"`
	RoundID       string `json:"round_id"`
	Error         string `json:"error"`
}

type ScoreModuleNotificationErrorPayload struct {
	CorrelationID string `json:"correlation_id"`
	RoundID       string `json:"round_id"`
	Error         string `json:"error"`
}

// --- Round Reminders ---
type RoundReminderPayload struct {
	RoundID      string `json:"round_id"`
	ReminderType string `json:"reminder_type"` // e.g., "one_hour", "thirty_minutes"
}

// --- Round State ---
type RoundStateUpdatedPayload struct {
	RoundID string                `json:"round_id"`
	State   roundtypes.RoundState `json:"state"`
}

// --- Start Round ---
type RoundStartedPayload struct {
	RoundID      string                `json:"round_id"`
	State        roundtypes.RoundState `json:"state"`
	Participants []Participant         `json:"participants"`
}

// Participant represents a participant in a round with their tag number.
type Participant struct {
	DiscordID string `json:"discord_id"`
	TagNumber int    `json:"tag_number"`
}

// --- Tag Retrieval ---
type TagNumberRequestPayload struct {
	DiscordID string        `json:"discord_id"`
	Timeout   time.Duration `json:"timeout"`
}

type RoundTagNumberFoundPayload struct {
	RoundID   string `json:"round_id"`
	DiscordID string `json:"discord_id"`
	TagNumber int    `json:"tag_number"`
}

type RoundTagNumberNotFoundPayload struct {
	DiscordID string `json:"discord_id"`
}

// --- Notify Score Module ---
type RoundScoresNotificationPayload struct {
	RoundID string             `json:"round_id"`
	Scores  []ParticipantScore `json:"scores"`
}

type ParticipantScore struct {
	DiscordID string  `json:"discord_id"`
	TagNumber string  `json:"tag_number"` // Assuming you want to keep this as a string
	Score     float64 `json:"score"`
}

// --- Process Round Scores ---
type ProcessRoundScoresRequestPayload struct {
	RoundID string             `json:"round_id"`
	Scores  []ParticipantScore `json:"scores"`
}

// --- User Authorization ---
type UserRoleCheckRequestPayload struct {
	DiscordID     string `json:"discord_id"`
	RoundID       string `json:"round_id"`       // Context for the request
	CorrelationID string `json:"correlation_id"` // To correlate with the response
}

type UserRoleCheckResultPayload struct {
	DiscordID string `json:"discord_id"`
	RoundID   string `json:"round_id"` // Context for the request
	HasRole   bool   `json:"has_role"`
	Error     string `json:"error"` // Error message if the check failed
}

type RoundUserRoleCheckErrorPayload struct {
	CorrelationID string `json:"correlation_id"`
	DiscordID     string `json:"discord_id"`
	RoundID       string `json:"round_id"`
	Error         string `json:"error"`
}

// --- Payloads for Tag Retrieval ---

type GetTagNumberResponsePayload struct {
	DiscordID string `json:"discord_id"`
	TagNumber int    `json:"tag_number"`
	Error     string `json:"error,omitempty"` // Include an error string
}
