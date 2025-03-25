package rounddb

import (
	"context"
	"time"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
)

// RoundDBInterface is the interface for interacting with the rounds database.
type RoundDB interface {
	CreateRound(ctx context.Context, round *roundtypes.Round) error
	GetRound(ctx context.Context, roundID roundtypes.ID) (*roundtypes.Round, error)
	UpdateRound(ctx context.Context, roundID roundtypes.ID, round *roundtypes.Round) error
	DeleteRound(ctx context.Context, roundID roundtypes.ID) error
	UpdateParticipant(ctx context.Context, roundID roundtypes.ID, participant roundtypes.Participant) ([]roundtypes.Participant, error)
	UpdateRoundState(ctx context.Context, roundID roundtypes.ID, state roundtypes.RoundState) error
	GetUpcomingRounds(ctx context.Context, startTime time.Time, endTime time.Time) ([]*roundtypes.Round, error)
	UpdateParticipantScore(ctx context.Context, roundID roundtypes.ID, participantID string, score int) error
	GetParticipantsWithResponses(ctx context.Context, roundID roundtypes.ID, responses ...string) ([]roundtypes.Participant, error)
	GetRoundState(ctx context.Context, roundID roundtypes.ID) (roundtypes.RoundState, error)
	LogRound(ctx context.Context, round *roundtypes.Round) error
	GetParticipants(ctx context.Context, roundID roundtypes.ID) ([]roundtypes.Participant, error)
	UpdateEventMessageID(ctx context.Context, roundID roundtypes.ID, eventMessageID string) error
	GetParticipant(ctx context.Context, roundID roundtypes.ID, userID usertypes.DiscordID) (*roundtypes.Participant, error)
	RemoveParticipant(ctx context.Context, roundID roundtypes.ID, userID usertypes.DiscordID) error
	GetEventMessageID(ctx context.Context, roundID roundtypes.ID) (*roundtypes.EventMessageID, error)
}
