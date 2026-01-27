package userservice

import (
	"context"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
)

// SUCCESS: *usertypes.UserData (Application/Shared type)
// FAILURE: error
type UserResult = results.OperationResult[*usertypes.UserData, error]

// If you need membership details, define a small struct here or in usertypes
type UserWithMembership struct {
	usertypes.UserData
	IsMember bool `json:"is_member"`
}

type UserWithMembershipResult = results.OperationResult[*UserWithMembership, error]

type Service interface {
	// User Creation
	CreateUser(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, tag *sharedtypes.TagNumber, udiscUsername *string, udiscName *string) (UserResult, error)
	// User Role
	UpdateUserRoleInDatabase(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, newRole sharedtypes.UserRoleEnum) (results.OperationResult[bool, error], error)
	// User Retrieval
	GetUserRole(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (results.OperationResult[sharedtypes.UserRoleEnum, error], error)
	GetUser(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (results.OperationResult[*UserWithMembership, error], error)
	UpdateUDiscIdentity(ctx context.Context, userID sharedtypes.DiscordID, username *string, name *string) (results.OperationResult[bool, error], error)
	// FindByUDiscUsername(ctx context.Context, guildID sharedtypes.GuildID, username string) (UserWithMembershipResult, error)
	// FindByUDiscName(ctx context.Context, guildID sharedtypes.GuildID, name string) (UserWithMembershipResult, error)
	// UDisc Matching
	// Accepts domain-specific player names slice instead of the round event payload.
	MatchParsedScorecard(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, playerNames []string) (results.OperationResult[*MatchResult, error], error)
}
