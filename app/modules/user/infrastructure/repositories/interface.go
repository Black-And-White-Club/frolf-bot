package userdb

import (
	"context"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// UserDB is an interface for interacting with the user database.
type UserDB interface {
	// WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
	CreateUser(ctx context.Context, user *User) error
	GetUserByUserID(ctx context.Context, userID sharedtypes.DiscordID) (*User, error)
	GetUserRole(ctx context.Context, userID sharedtypes.DiscordID) (sharedtypes.UserRoleEnum, error)
	UpdateUserRole(ctx context.Context, userID sharedtypes.DiscordID, role sharedtypes.UserRoleEnum) error
}
