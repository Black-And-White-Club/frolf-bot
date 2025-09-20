package rounddb

import (
	"context"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/uptrace/bun"
)

// RoundDBInterface is the interface for interacting with the rounds database.
type RoundDB interface {
	CreateRound(ctx context.Context, guildID sharedtypes.GuildID, round *roundtypes.Round) error
	GetRound(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (*roundtypes.Round, error)
	UpdateRound(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, round *roundtypes.Round) (*roundtypes.Round, error)
	DeleteRound(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) error
	UpdateParticipant(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, participant roundtypes.Participant) ([]roundtypes.Participant, error)
	UpdateRoundState(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, state roundtypes.RoundState) error
	GetUpcomingRounds(ctx context.Context, guildID sharedtypes.GuildID) ([]*roundtypes.Round, error)
	UpdateParticipantScore(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, participantID sharedtypes.DiscordID, score sharedtypes.Score) error
	GetParticipantsWithResponses(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, responses ...string) ([]roundtypes.Participant, error)
	GetRoundState(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (roundtypes.RoundState, error)
	LogRound(ctx context.Context, guildID sharedtypes.GuildID, round *roundtypes.Round) error
	GetParticipants(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) ([]roundtypes.Participant, error)
	UpdateEventMessageID(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, eventMessageID string) (*roundtypes.Round, error)
	GetParticipant(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, userID sharedtypes.DiscordID) (*roundtypes.Participant, error)
	RemoveParticipant(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, userID sharedtypes.DiscordID) ([]roundtypes.Participant, error)
	GetEventMessageID(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (string, error)
	UpdateRoundsAndParticipants(ctx context.Context, guildID sharedtypes.GuildID, updates []roundtypes.RoundUpdate) error
	TagUpdates(ctx context.Context, guildID sharedtypes.GuildID, bun bun.IDB, round *roundtypes.Round) error
}
