package userservice

import (
	"context"

	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	"github.com/ThreeDotsLabs/watermill/message"
)

type Service interface {
	// User Creation
	CreateUser(ctx context.Context, msg *message.Message, discordID usertypes.DiscordID, tag *int) error
	PublishUserCreated(ctx context.Context, msg *message.Message, discordID usertypes.DiscordID, tag *int) error
	PublishUserCreationFailed(ctx context.Context, msg *message.Message, discordID usertypes.DiscordID, tag *int, reason string) error

	// User Role
	UpdateUserRole(ctx context.Context, msg *message.Message, discordID usertypes.DiscordID, role usertypes.UserRoleEnum, requesterID usertypes.DiscordID) error
	UpdateUserRoleInDatabase(context.Context, *message.Message, usertypes.DiscordID, usertypes.UserRoleEnum) error
	PublishUserRoleUpdated(ctx context.Context, msg *message.Message, discordID usertypes.DiscordID, role usertypes.UserRoleEnum) error
	PublishUserRoleUpdateFailed(ctx context.Context, msg *message.Message, discordID usertypes.DiscordID, role usertypes.UserRoleEnum, reason string) error
	// User Retrieval
	GetUserRole(ctx context.Context, msg *message.Message, dicscordID usertypes.DiscordID) error
	GetUser(ctx context.Context, msg *message.Message, discordID usertypes.DiscordID) error

	// User Permissions
	CheckUserPermissions(ctx context.Context, msg *message.Message, userID usertypes.DiscordID, role usertypes.UserRoleEnum, requesterID usertypes.DiscordID) error
	CheckUserPermissionsInDB(ctx context.Context, msg *message.Message, discordID usertypes.DiscordID, role usertypes.UserRoleEnum, requesterID usertypes.DiscordID) error
	PublishUserPermissionsCheckResponse(ctx context.Context, msg *message.Message, discordID usertypes.DiscordID, role usertypes.UserRoleEnum, requesterID usertypes.DiscordID, hasPermission bool, reason string) error
	PublishUserPermissionsCheckFailed(ctx context.Context, msg *message.Message, discordID usertypes.DiscordID, role usertypes.UserRoleEnum, requesterID usertypes.DiscordID, reason string) error

	// Tag Availability
	CheckTagAvailability(ctx context.Context, msg *message.Message, tagNumber int, discordID usertypes.DiscordID) error
	TagUnavailable(ctx context.Context, msg *message.Message, tagNumber int, discordID usertypes.DiscordID) error
}
