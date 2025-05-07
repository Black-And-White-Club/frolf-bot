package rounddb

import (
	"context"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/uptrace/bun"
)

// RoundDBInterface is the interface for interacting with the rounds database.
type RoundDB interface {
	CreateRound(ctx context.Context, round *roundtypes.Round) error
	GetRound(ctx context.Context, roundID sharedtypes.RoundID) (*roundtypes.Round, error)
	UpdateRound(ctx context.Context, roundID sharedtypes.RoundID, round *roundtypes.Round) error
	DeleteRound(ctx context.Context, roundID sharedtypes.RoundID) error
	UpdateParticipant(ctx context.Context, roundID sharedtypes.RoundID, participant roundtypes.Participant) ([]roundtypes.Participant, error)
	UpdateRoundState(ctx context.Context, roundID sharedtypes.RoundID, state roundtypes.RoundState) error
	GetUpcomingRounds(ctx context.Context) ([]*roundtypes.Round, error)
	UpdateParticipantScore(ctx context.Context, roundID sharedtypes.RoundID, participantID sharedtypes.DiscordID, score sharedtypes.Score) error
	GetParticipantsWithResponses(ctx context.Context, roundID sharedtypes.RoundID, responses ...string) ([]roundtypes.Participant, error)
	GetRoundState(ctx context.Context, roundID sharedtypes.RoundID) (roundtypes.RoundState, error)
	LogRound(ctx context.Context, round *roundtypes.Round) error
	GetParticipants(ctx context.Context, roundID sharedtypes.RoundID) ([]roundtypes.Participant, error)
	UpdateEventMessageID(ctx context.Context, roundID sharedtypes.RoundID, eventMessageID string) (*roundtypes.Round, error)
	GetParticipant(ctx context.Context, roundID sharedtypes.RoundID, userID sharedtypes.DiscordID) (*roundtypes.Participant, error)
	RemoveParticipant(ctx context.Context, roundID sharedtypes.RoundID, userID sharedtypes.DiscordID) error
	GetEventMessageID(ctx context.Context, roundID sharedtypes.RoundID) (*sharedtypes.RoundID, error)
	UpdateRoundsAndParticipants(ctx context.Context, updates []roundtypes.RoundUpdate) error
	TagUpdates(ctx context.Context, bun bun.IDB, round *roundtypes.Round) error
}
