package rounddb

import (
	"context"
	"time"
)

// RoundDB is the interface for interacting with the rounds database.
type RoundDBInterface interface {
	CreateRound(ctx context.Context, round *Round) error
	GetRound(ctx context.Context, roundID int64) (*Round, error)
	UpdateRound(ctx context.Context, roundID int64, round *Round) error
	DeleteRound(ctx context.Context, roundID int64) error
	UpdateParticipant(ctx context.Context, roundID int64, participant Participant) error
	UpdateRoundState(ctx context.Context, roundID int64, state RoundState) error
	GetUpcomingRounds(ctx context.Context, startTime time.Time, endTime time.Time) ([]*Round, error)
	UpdateParticipantScore(ctx context.Context, roundID int64, participantID string, score int) error
	GetParticipantsWithResponses(ctx context.Context, roundID int64, responses ...string) ([]Participant, error)
	GetRoundState(ctx context.Context, roundID int64) (RoundState, error)
	LogRound(ctx context.Context, round *Round) error
	GetParticipants(ctx context.Context, roundID int64) ([]Participant, error)
	UpdateDiscordEventID(ctx context.Context, roundID int64, discordEventID string) error
}
