package rounddb

import (
	"context"
	"time"

	rounddto "github.com/Black-And-White-Club/tcr-bot/app/modules/round/dto"
)

// RoundDB is the interface for interacting with the rounds database.
type RoundDB interface {
	GetRounds(ctx context.Context) ([]*Round, error)
	GetRound(ctx context.Context, roundID int64) (*Round, error)
	CreateRound(ctx context.Context, input rounddto.CreateRoundInput) (*Round, error)
	CreateRoundScores(ctx context.Context, roundID int64, scores map[string]int) error
	UpdateRound(ctx context.Context, roundID int64, updates map[string]interface{}) error
	DeleteRound(ctx context.Context, roundID int64) error
	UpdateParticipant(ctx context.Context, roundID int64, participant Participant) error
	UpdateRoundState(ctx context.Context, roundID int64, state RoundState) error
	GetUpcomingRounds(ctx context.Context, now, oneHourFromNow time.Time) ([]*Round, error)
	SubmitScore(ctx context.Context, roundID int64, discordID string, score int) error
	IsRoundFinalized(ctx context.Context, roundID int64) (bool, error)
	GetRoundState(ctx context.Context, roundID int64) (RoundState, error)
	RecordScores(ctx context.Context, roundID int64, scores map[string]int) error
	RoundExists(ctx context.Context, roundID int64) (bool, error)
	GetParticipant(ctx context.Context, roundID int64, discordID string) (Participant, error)
	GetScoreForParticipant(ctx context.Context, roundID int64, participantID string) (*Score, error)
	UpdatePendingScores(ctx context.Context, roundID int64, pendingScores []Score) error
}
