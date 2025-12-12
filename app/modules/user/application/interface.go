package userservice

import (
	"context"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

type Service interface {
	// User Creation
	CreateUser(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, tag *sharedtypes.TagNumber, udiscUsername *string, udiscName *string) (UserOperationResult, error)

	// User Role
	UpdateUserRoleInDatabase(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, newRole sharedtypes.UserRoleEnum) (UserOperationResult, error)

	// User Retrieval
	GetUserRole(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (UserOperationResult, error)
	GetUser(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (UserOperationResult, error)

	// UDisc Identity Management
	UpdateUDiscIdentity(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, username *string, name *string) (UserOperationResult, error)
	FindByUDiscUsername(ctx context.Context, guildID sharedtypes.GuildID, username string) (UserOperationResult, error)
	FindByUDiscName(ctx context.Context, guildID sharedtypes.GuildID, name string) (UserOperationResult, error)

	// UDisc Matching
	MatchParsedScorecard(ctx context.Context, payload roundevents.ParsedScorecardPayload) (UserOperationResult, error)
}
