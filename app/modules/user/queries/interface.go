package userqueries

import (
	"context"

	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/db"
)

// QueryService defines the interface for user queries.
type QueryService interface {
	GetUserByDiscordID(ctx context.Context, discordID string) (*userdb.User, error)
	GetUserRole(ctx context.Context, discordID string) (userdb.UserRole, error)
}
