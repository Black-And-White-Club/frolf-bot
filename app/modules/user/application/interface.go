package userservice

import (
	"context"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

type Service interface {
	// User Creation
	CreateUser(ctx context.Context, userID sharedtypes.DiscordID, tag *sharedtypes.TagNumber) (UserOperationResult, error)

	// User Role
	UpdateUserRoleInDatabase(ctx context.Context, userID sharedtypes.DiscordID, newRole sharedtypes.UserRoleEnum) (UserOperationResult, error)

	// User Retrieval
	GetUserRole(ctx context.Context, userID sharedtypes.DiscordID) (UserOperationResult, error)
	GetUser(ctx context.Context, userID sharedtypes.DiscordID) (UserOperationResult, error)
}
