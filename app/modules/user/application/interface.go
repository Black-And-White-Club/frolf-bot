package userservice

import (
	"context"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

type Service interface {
	// User Creation
	CreateUser(ctx context.Context, userID sharedtypes.DiscordID, tag *sharedtypes.TagNumber) (*userevents.UserCreatedPayload, *userevents.UserCreationFailedPayload, error)

	// User Role
	UpdateUserRoleInDatabase(ctx context.Context, userID sharedtypes.DiscordID, newRole sharedtypes.UserRoleEnum) (*userevents.UserRoleUpdateResultPayload, *userevents.UserRoleUpdateFailedPayload, error)

	// User Retrieval
	GetUserRole(ctx context.Context, userID sharedtypes.DiscordID) (*userevents.GetUserRoleResponsePayload, *userevents.GetUserRoleFailedPayload, error)
	GetUser(ctx context.Context, userID sharedtypes.DiscordID) (*userevents.GetUserResponsePayload, *userevents.GetUserFailedPayload, error)
}
