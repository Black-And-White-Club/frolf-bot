package rounddb

import (
	"context"
	"time"

	roundtypes "github.com/Black-And-White-Club/frolf-bot/app/modules/round/domain/types"
)

// RoundDB is the interface for interacting with the rounds database.
type RoundDB interface {
	GetParticipantsWithResponses(ctx context.Context, roundID string, responses ...roundtypes.Response) ([]roundtypes.RoundParticipant, error) // Updated return type
	CreateRound(ctx context.Context, round *roundtypes.Round) error                                                                            // Updated to use roundtypes.Round
	GetRound(ctx context.Context, roundID string) (*roundtypes.Round, error)                                                                   // Updated to use roundtypes.Round
	GetRoundState(ctx context.Context, roundID string) (roundtypes.RoundState, error)                                                          // Updated to use roundtypes.RoundState
	UpdateRound(ctx context.Context, roundID string, round *roundtypes.Round) error                                                            // Updated to use roundtypes.Round
	DeleteRound(ctx context.Context, roundID string) error
	LogRound(ctx context.Context, round *roundtypes.Round) error                                          // Updated to use roundtypes.Round
	UpdateParticipant(ctx context.Context, roundID string, participant roundtypes.RoundParticipant) error // Updated to use roundtypes.RoundParticipant
	UpdateRoundState(ctx context.Context, roundID string, state roundtypes.RoundState) error              // Updated to use roundtypes.RoundState
	GetUpcomingRounds(ctx context.Context, now, oneHourFromNow time.Time) ([]*roundtypes.Round, error)    // Updated to use roundtypes.Round
	UpdateParticipantScore(ctx context.Context, roundID string, participantID string, score int) error
	GetParticipants(ctx context.Context, roundID string) ([]roundtypes.RoundParticipant, error) // Added to get participants for a round
}
