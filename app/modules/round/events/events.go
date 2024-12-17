package roundhandlers

import (
	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
	rounddto "github.com/Black-And-White-Club/tcr-bot/app/modules/round/dto"
)

// FinalizeRoundEvent represents the event to finalize a round.
type FinalizeRoundEvent struct {
	RoundID int64 `json:"round_id"`
}

// GetTagNumberResponse represents the response to a GetTagNumberRequest.
type GetTagNumberResponse struct {
	RoundID   int64  `json:"round_id"`
	DiscordID string `json:"discord_id"`
	TagNumber *int   `json:"tag_number"`
}

// ParticipantJoinedRoundEvent represents the event published when a participant joins a round.
type ParticipantJoinedRoundEvent struct {
	RoundID     int64               `json:"round_id"`
	Participant rounddb.Participant `json:"participant"`
}

// SendScoresEvent represents the event to send scores to the score module.
type SendScoresEvent struct {
	RoundID int64              `json:"round_id"`
	Scores  []ParticipantScore `json:"scores"`
}

// ParticipantScore represents the score information for a participant.
type ParticipantScore struct {
	DiscordID string `json:"discord_id"`
	TagNumber int    `json:"tag_number"`
	Score     int    `json:"score"`
}

type RoundCreatedEvent struct {
	RoundID int64                     `json:"round_id"`
	Input   rounddto.CreateRoundInput `json:"input"`
}
type RoundDeletedEvent struct {
	RoundID int64 `json:"round_id"`
}

// RoundEditedEvent represents the event triggered when a round is edited.
type RoundEditedEvent struct {
	RoundID int64                  `json:"round_id"`
	Updates map[string]interface{} `json:"updates"`
}

type RoundReminderEvent struct {
	RoundID      int64  `json:"round_id"`
	ReminderType string `json:"reminder_type"`
}

// RoundScoresProcessed represents the event of scores being processed for a round.
type RoundScoresProcessedEvent struct {
	RoundID int64          `json:"round_id"`
	Scores  map[string]int `json:"scores"`
}

type RoundStateUpdatedEvent struct {
	RoundID int64              `json:"round_id"`
	State   rounddb.RoundState `json:"state"` // Use the db model for RoundState
}

type RoundStartEvent struct {
	RoundID int64 `json:"round_id"` // The ID of the round to start
}
