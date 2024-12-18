package userservice

import (
	"context"

	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/db"
	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/events"
)

// UserService handles user-related logic.
type Service interface {
	OnUserSignupRequest(ctx context.Context, req userevents.UserSignupRequest) (*userevents.UserSignupResponse, error)
	OnUserRoleUpdateRequest(ctx context.Context, req userevents.UserRoleUpdateRequest) (*userevents.UserRoleUpdateResponse, error)
	GetUserRole(ctx context.Context, discordID string) (*userdb.UserRole, error)
	GetUser(ctx context.Context, discordID string) (*userdb.User, error)
}
