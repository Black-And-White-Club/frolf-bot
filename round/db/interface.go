package rounddb

import (
	"context"
	"time"
)

// RoundDB is the interface for interacting with the rounds database.
type RoundDB interface {
	GetRounds(ctx context.Context) ([]*Round, error)
	GetRound(ctx context.Context, roundID int64) (*Round, error)
	CreateRound(ctx context.Context, round ScheduleRoundInput) (*Round, error)
	UpdateRound(ctx context.Context, roundID int64, input EditRoundInput) error
	DeleteRound(ctx context.Context, roundID int64) error
	UpdateParticipant(ctx context.Context, roundID int64, participant Participant) error
	UpdateRoundState(ctx context.Context, roundID int64, state RoundState) error
	GetUpcomingRounds(ctx context.Context, now, oneHourFromNow time.Time) ([]*Round, error)
	SubmitScore(ctx context.Context, roundID int64, discordID string, score int) error
	IsRoundFinalized(ctx context.Context, roundID int64) (bool, error)
	IsUserParticipant(ctx context.Context, roundID int64, DiscordID string) (bool, error)
	GetRoundState(ctx context.Context, roundID int64) (RoundState, error)
	RoundExists(ctx context.Context, roundID int64) (bool, error)
}
