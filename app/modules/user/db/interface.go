package userdb

import (
	"context"
)

// UserDB is an interface for interacting with the user database.
type UserDB interface {
	CreateUser(ctx context.Context, user *User) error
	GetUserByDiscordID(ctx context.Context, discordID string) (*User, error)
	UpdateUser(ctx context.Context, discordID string, updates map[string]interface{}) error
	GetUserRole(ctx context.Context, discordID string) (UserRole, error)
	UpdateUserRole(ctx context.Context, discordID string, newRole UserRole) error
}
