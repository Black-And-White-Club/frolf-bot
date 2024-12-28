package rounddb

import (
	"context"
	"time"
)

// RoundDB is the interface for interacting with the rounds database.
type RoundDB interface {
	GetParticipantsWithResponses(ctx context.Context, roundID string, responses ...Response) ([]Participant, error)
	CreateRound(ctx context.Context, round *Round) error
	GetRound(ctx context.Context, roundID string) (*Round, error)
	GetRoundState(ctx context.Context, roundID string) (RoundState, error)
	UpdateRound(ctx context.Context, roundID string, round *Round) error
	DeleteRound(ctx context.Context, roundID string) error
	LogRound(ctx context.Context, round *Round, updateType ScoreUpdateType) error
	UpdateParticipant(ctx context.Context, roundID string, participant Participant) error
	UpdateRoundState(ctx context.Context, roundID string, state RoundState) error
	GetUpcomingRounds(ctx context.Context, now, oneHourFromNow time.Time) ([]*Round, error)
	UpdateParticipantScore(ctx context.Context, roundID, participantID string, score int) error
}
