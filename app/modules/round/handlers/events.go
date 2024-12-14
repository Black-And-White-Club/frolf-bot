package roundhandlers

import (
	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
	rounddto "github.com/Black-And-White-Club/tcr-bot/app/modules/round/dto"
)

// ParticipantJoinedRoundEvent represents the event published when a participant joins a round.
type ParticipantJoinedRoundEvent struct {
	RoundID     int64               `json:"round_id"`
	Participant rounddb.Participant `json:"participant"`
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

type RoundStateUpdatedEvent struct {
	RoundID int64              `json:"round_id"`
	State   rounddb.RoundState `json:"state"` // Use the db model for RoundState
}
