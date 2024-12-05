// internal/db/user.go

package db

import (
	"context"

	"github.com/Black-And-White-Club/tcr-bot/app/models"
)

// UserDB is an interface for user-related database operations.
type UserDB interface {
	GetUser(ctx context.Context, discordID string) (*models.User, error)
	CreateUser(ctx context.Context, user *models.User) error
	UpdateUser(ctx context.Context, discordID string, updates *models.User) error
	Ping(ctx context.Context) error
}
