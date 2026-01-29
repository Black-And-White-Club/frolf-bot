package userservice

import (
	"context"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
)

// CreateUserResponse contains the outcome of a user creation operation.
type CreateUserResponse struct {
	usertypes.UserData
	TagNumber       *sharedtypes.TagNumber `json:"tag_number,omitempty"`
	IsReturningUser bool                   `json:"is_returning_user"`
}

// SUCCESS: *CreateUserResponse
// FAILURE: error
type UserResult = results.OperationResult[*CreateUserResponse, error]

// If you need membership details, define a small struct here or in usertypes
type UserWithMembership struct {
	usertypes.UserData
	UDiscUsername *string `json:"udisc_username"`
	UDiscName     *string `json:"udisc_name"`
	IsMember      bool    `json:"is_member"`
}

type UserWithMembershipResult = results.OperationResult[*UserWithMembership, error]

type UserRoleResult = results.OperationResult[sharedtypes.UserRoleEnum, error]

type UpdateIdentityResult = results.OperationResult[bool, error]

// MatchResult holds the domain outcome of a scorecard matching operation.
type MatchResult struct {
	Mappings  []userevents.UDiscConfirmedMappingV1
	Unmatched []string
}

type MatchResultResult = results.OperationResult[*MatchResult, error]

type Service interface {
	// User Creation
	CreateUser(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, tag *sharedtypes.TagNumber, udiscUsername *string, udiscName *string) (UserResult, error)
	// User Role
	UpdateUserRoleInDatabase(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, newRole sharedtypes.UserRoleEnum) (UpdateIdentityResult, error)
	// User Retrieval
	GetUserRole(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (UserRoleResult, error)
	GetUser(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID) (UserWithMembershipResult, error)
	FindByUDiscUsername(ctx context.Context, guildID sharedtypes.GuildID, username string) (UserWithMembershipResult, error)
	FindByUDiscName(ctx context.Context, guildID sharedtypes.GuildID, name string) (UserWithMembershipResult, error)
	UpdateUDiscIdentity(ctx context.Context, userID sharedtypes.DiscordID, username *string, name *string) (UpdateIdentityResult, error)
	// UDisc Matching
	MatchParsedScorecard(ctx context.Context, guildID sharedtypes.GuildID, userID sharedtypes.DiscordID, playerNames []string) (MatchResultResult, error)
}
