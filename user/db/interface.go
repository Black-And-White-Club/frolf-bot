// user/db/interface.go
package userdb

import (
	"context"
)

// UserDB is an interface for interacting with the user database.
type UserDB interface {
	CreateUser(ctx context.Context, user *User) error
	GetUser(ctx context.Context, discordID string) (*User, error)
	UpdateUser(ctx context.Context, discordID string, updates *User) error
}
