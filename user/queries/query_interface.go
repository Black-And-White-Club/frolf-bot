// user/queries/query_interface.go
package queries

import (
	"context"

	"github.com/Black-And-White-Club/tcr-bot/user/db"
)

// QueryService defines the interface for user queries.
type QueryService interface {
	GetUserByID(ctx context.Context, discordID string) (*db.User, error)
	GetUserRole(ctx context.Context, discordID string) (string, error)
	// Add other query methods as needed (e.g., GetAllUsers)
}
