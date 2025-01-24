package userservice

import (
	"context"

	usertypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types"
	"github.com/ThreeDotsLabs/watermill/message"
)

type Service interface {
	// User Creation
	CreateUser(ctx context.Context, msg *message.Message, discordID usertypes.DiscordID, tag *int) error
	PublishUserCreated(ctx context.Context, msg *message.Message, discordID usertypes.DiscordID, tag *int) error
	PublishUserCreationFailed(ctx context.Context, msg *message.Message, discordID usertypes.DiscordID, tag *int, reason string) error

	// User Role
	UpdateUserRole(ctx context.Context, msg *message.Message, userID usertypes.DiscordID, role, requesterID string) error
	UpdateUserRoleInDatabase(ctx context.Context, msg *message.Message, userID string, role string) error
	PublishUserRoleUpdated(ctx context.Context, msg *message.Message, userID, role string) error
	PublishUserRoleUpdateFailed(ctx context.Context, msg *message.Message, userID, role, reason string) error

	// User Retrieval
	GetUserRole(ctx context.Context, msg *message.Message, discordID usertypes.DiscordID) error
	GetUser(ctx context.Context, msg *message.Message, discordID usertypes.DiscordID) error

	// User Permissions
	CheckUserPermissions(ctx context.Context, msg *message.Message, userID usertypes.DiscordID, role usertypes.UserRoleEnum, requesterID string) error
	CheckUserPermissionsInDB(ctx context.Context, msg *message.Message, discordID usertypes.DiscordID, role usertypes.UserRoleEnum, requesterID string) error
	PublishUserPermissionsCheckResponse(ctx context.Context, msg *message.Message, discordID usertypes.DiscordID, role usertypes.UserRoleEnum, requesterID string, hasPermission bool, reason string) error
	PublishUserPermissionsCheckFailed(ctx context.Context, msg *message.Message, discordID usertypes.DiscordID, role usertypes.UserRoleEnum, requesterID, reason string) error

	// Tag Availability
	CheckTagAvailability(ctx context.Context, msg *message.Message, tagNumber int) error
}
