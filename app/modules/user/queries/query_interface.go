package userqueries

import (
	"context"

	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/db"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
)

// QueryService defines the interface for user queries.
type QueryService interface {
	GetUserByDiscordID(ctx context.Context, discordID string) (*userdb.User, error)
	GetUserRole(ctx context.Context, discordID string) (string, error)
	// Add other query methods as needed (e.g., GetAllUsers)

	// Update the EventBus method to return your PubSub type
	EventBus() *watermillutil.PubSub
}
