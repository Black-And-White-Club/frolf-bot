package rounddb

import (
	"context"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// Repository defines the contract for round persistence.
// All methods are context-aware for cancellation and timeout propagation.
//
// Error semantics:
//   - ErrNotFound: Record does not exist (GetRound, GetParticipant)
//   - ErrNoRowsAffected: UPDATE/DELETE matched no rows
//   - ErrParticipantNotFound: Participant not in round (UpdateParticipantScore)
//   - Other errors: Infrastructure failures (DB connection, query errors)
type Repository interface {
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
	GetParticipants(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) ([]roundtypes.Participant, error)
	UpdateEventMessageID(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, eventMessageID string) (*roundtypes.Round, error)
	GetParticipant(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, userID sharedtypes.DiscordID) (*roundtypes.Participant, error)
	RemoveParticipant(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, userID sharedtypes.DiscordID) ([]roundtypes.Participant, error)
	GetEventMessageID(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (string, error)
	UpdateRoundsAndParticipants(ctx context.Context, guildID sharedtypes.GuildID, updates []roundtypes.RoundUpdate) error
	GetUpcomingRoundsByParticipant(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) ([]*roundtypes.Round, error)
	UpdateImportStatus(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, importID string, status string, errorMessage string, errorCode string) error
	CreateRoundGroups(ctx context.Context, roundID sharedtypes.RoundID, participants []roundtypes.Participant) error
	RoundHasGroups(ctx context.Context, roundID sharedtypes.RoundID) (bool, error)
	GetRoundsByGuildID(ctx context.Context, guildID sharedtypes.GuildID, states ...roundtypes.RoundState) ([]*roundtypes.Round, error)
}
