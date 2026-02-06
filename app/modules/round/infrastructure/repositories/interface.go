package rounddb

import (
	"context"

	"github.com/uptrace/bun"

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
	CreateRound(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, round *roundtypes.Round) error
	GetRound(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (*roundtypes.Round, error)
	UpdateRound(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, round *roundtypes.Round) (*roundtypes.Round, error)
	DeleteRound(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) error
	UpdateParticipant(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, participant roundtypes.Participant) ([]roundtypes.Participant, error)
	UpdateRoundState(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, state roundtypes.RoundState) error
	GetUpcomingRounds(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) ([]*roundtypes.Round, error)
	UpdateParticipantScore(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, participantID sharedtypes.DiscordID, score sharedtypes.Score) error
	GetParticipantsWithResponses(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, responses ...string) ([]roundtypes.Participant, error)
	GetRoundState(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (roundtypes.RoundState, error)
	GetParticipants(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) ([]roundtypes.Participant, error)
	UpdateEventMessageID(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, eventMessageID string) (*roundtypes.Round, error)
	UpdateDiscordEventID(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, discordEventID string) (*roundtypes.Round, error)
	GetRoundByDiscordEventID(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, discordEventID string) (*roundtypes.Round, error)
	GetParticipant(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, userID sharedtypes.DiscordID) (*roundtypes.Participant, error)
	RemoveParticipant(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, userID sharedtypes.DiscordID) ([]roundtypes.Participant, error)
	GetEventMessageID(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (string, error)
	UpdateRoundsAndParticipants(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, updates []roundtypes.RoundUpdate) error
	GetUpcomingRoundsByParticipant(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) ([]*roundtypes.Round, error)
	UpdateImportStatus(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID, importID string, status string, errorMessage string, errorCode string) error
	CreateRoundGroups(ctx context.Context, db bun.IDB, roundID sharedtypes.RoundID, participants []roundtypes.Participant) error
	RoundHasGroups(ctx context.Context, db bun.IDB, roundID sharedtypes.RoundID) (bool, error)
	GetRoundsByGuildID(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID, states ...roundtypes.RoundState) ([]*roundtypes.Round, error)
}
