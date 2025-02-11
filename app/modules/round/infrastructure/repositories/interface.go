package rounddb

import (
	"context"
	"time"

	roundtypes "github.com/Black-And-White-Club/frolf-bot/app/modules/round/domain/types"
)

// RoundDB is the interface for interacting with the rounds database.
type RoundDB interface {
	GetParticipantsWithResponses(ctx context.Context, roundID string, responses ...roundtypes.Response) ([]roundtypes.RoundParticipant, error)
	CreateRound(ctx context.Context, round *roundtypes.Round) error
	GetRound(ctx context.Context, roundID string) (*roundtypes.Round, error)
	GetRoundState(ctx context.Context, roundID string) (roundtypes.RoundState, error)
	UpdateRound(ctx context.Context, roundID string, round *roundtypes.Round) error
	DeleteRound(ctx context.Context, roundID string) error
	LogRound(ctx context.Context, round *roundtypes.Round) error
	UpdateParticipant(ctx context.Context, roundID string, participant roundtypes.RoundParticipant) error
	UpdateRoundState(ctx context.Context, roundID string, state roundtypes.RoundState) error
	GetUpcomingRounds(ctx context.Context, now, oneHourFromNow time.Time) ([]*roundtypes.Round, error)
	UpdateParticipantScore(ctx context.Context, roundID string, participantID string, score int) error
	GetParticipants(ctx context.Context, roundID string) ([]roundtypes.RoundParticipant, error)
	UpdateDiscordEventID(ctx context.Context, roundID string, discordEventID string) error
}
