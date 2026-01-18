package userservice

import (
	"context"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
)

type Service interface {
	// User Creation
	CreateUser(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, tag *sharedtypes.TagNumber, udiscUsername *string, udiscName *string) (results.OperationResult, error)

	// User Role
	UpdateUserRoleInDatabase(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, newRole sharedtypes.UserRoleEnum) (results.OperationResult, error)

	// User Retrieval
	GetUserRole(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (results.OperationResult, error)
	GetUser(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (results.OperationResult, error)

	// UDisc Identity Management
	UpdateUDiscIdentity(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, username *string, name *string) (results.OperationResult, error)
	FindByUDiscUsername(ctx context.Context, guildID sharedtypes.GuildID, username string) (results.OperationResult, error)
	FindByUDiscName(ctx context.Context, guildID sharedtypes.GuildID, name string) (results.OperationResult, error)

	// UDisc Matching
	MatchParsedScorecard(ctx context.Context, payload roundevents.ParsedScorecardPayloadV1) (results.OperationResult, error)
}
