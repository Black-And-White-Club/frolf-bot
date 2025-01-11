package userservice

import (
	"context"

	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	usertypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types"
)

// Service defines the interface for user service operations.
type Service interface {
	// OnUserSignupRequest handles a user signup request.
	OnUserSignupRequest(ctx context.Context, req userevents.UserSignupRequestPayload) (*userevents.UserSignupResponsePayload, error)
	// OnUserRoleUpdateRequest handles a user role update request.
	OnUserRoleUpdateRequest(ctx context.Context, req userevents.UserRoleUpdateRequestPayload) (*userevents.UserRoleUpdateResponsePayload, error)
	// GetUserRole retrieves the role of a user.
	GetUserRole(ctx context.Context, discordID usertypes.DiscordID) (usertypes.UserRoleEnum, error)
	// GetUser retrieves user data by Discord ID.
	GetUser(ctx context.Context, discordID usertypes.DiscordID) (usertypes.User, error)
}
