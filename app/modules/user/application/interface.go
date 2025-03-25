package userservice

import (
	"context"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	"github.com/ThreeDotsLabs/watermill/message"
)

type Service interface {
	// User Creation
	CreateUser(ctx context.Context, msg *message.Message, userID usertypes.DiscordID, tag *int) (*userevents.UserCreatedPayload, *userevents.UserCreationFailedPayload, error)

	// User Role
	UpdateUserRoleInDatabase(ctx context.Context, msg *message.Message, userID usertypes.DiscordID, newRole usertypes.UserRoleEnum) (*userevents.UserRoleUpdateResultPayload, *userevents.UserRoleUpdateFailedPayload, error)

	// User Retrieval
	GetUserRole(ctx context.Context, msg *message.Message, userID usertypes.DiscordID) (*userevents.GetUserRoleResponsePayload, *userevents.GetUserRoleFailedPayload, error)
	GetUser(ctx context.Context, msg *message.Message, userID usertypes.DiscordID) (*userevents.GetUserResponsePayload, *userevents.GetUserFailedPayload, error)
}
