package userqueries

import (
	"context"

	userdb "github.com/Black-And-White-Club/tcr-bot/user/db"
	"github.com/Black-And-White-Club/tcr-bot/watermillcmd"
)

// QueryService defines the interface for user queries.
type QueryService interface {
	GetUserByDiscordID(ctx context.Context, discordID string) (*userdb.User, error)
	GetUserRole(ctx context.Context, discordID string) (string, error)
	// Add other query methods as needed (e.g., GetAllUsers)

	// Add the EventBus method
	EventBus() watermillcmd.MessageBus
}
