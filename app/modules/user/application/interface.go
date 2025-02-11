package userservice

import (
	"context"

	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	"github.com/ThreeDotsLabs/watermill/message"
)

type Service interface {
	// User Creation
	CreateUser(context.Context, *message.Message, usertypes.DiscordID, *int) error
	PublishUserCreated(context.Context, *message.Message, string, *int) error
	PublishUserCreationFailed(context.Context, *message.Message, usertypes.DiscordID, *int, string) error

	// User Role
	UpdateUserRole(context.Context, *message.Message, usertypes.DiscordID, usertypes.UserRoleEnum, string) error
	UpdateUserRoleInDatabase(context.Context, *message.Message, usertypes.DiscordID, usertypes.UserRoleEnum) error
	PublishUserRoleUpdated(context.Context, *message.Message, usertypes.DiscordID, usertypes.UserRoleEnum) error
	PublishUserRoleUpdateFailed(context.Context, *message.Message, usertypes.DiscordID, usertypes.UserRoleEnum, string) error

	// User Retrieval
	GetUserRole(ctx context.Context, msg *message.Message, discordID usertypes.DiscordID) error
	GetUser(ctx context.Context, msg *message.Message, discordID usertypes.DiscordID) error

	// User Permissions
	CheckUserPermissions(ctx context.Context, msg *message.Message, userID usertypes.DiscordID, role usertypes.UserRoleEnum, requesterID string) error
	CheckUserPermissionsInDB(ctx context.Context, msg *message.Message, discordID usertypes.DiscordID, role usertypes.UserRoleEnum, requesterID string) error
	PublishUserPermissionsCheckResponse(ctx context.Context, msg *message.Message, discordID usertypes.DiscordID, role usertypes.UserRoleEnum, requesterID string, hasPermission bool, reason string) error
	PublishUserPermissionsCheckFailed(ctx context.Context, msg *message.Message, discordID usertypes.DiscordID, role usertypes.UserRoleEnum, requesterID, reason string) error

	// Tag Availability
	CheckTagAvailability(ctx context.Context, msg *message.Message, tagNumber int, discordID usertypes.DiscordID) error
	TagUnavailable(ctx context.Context, msg *message.Message, tagNumber int, discordID usertypes.DiscordID) error
}
