package userservice

import (
	"context"

	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	usertypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types"
)

// Service handles user-related logic.
type Service interface {
	OnUserSignupRequest(ctx context.Context, req userevents.UserSignupRequestPayload) (*userevents.UserSignupResponsePayload, error)
	OnUserRoleUpdateRequest(ctx context.Context, req userevents.UserRoleUpdateRequestPayload) (*userevents.UserRoleUpdateResponsePayload, error)
	GetUserRole(ctx context.Context, discordID usertypes.DiscordID) (*usertypes.UserRoleEnum, error)
	GetUser(ctx context.Context, discordID usertypes.DiscordID) (*usertypes.User, error)
}
