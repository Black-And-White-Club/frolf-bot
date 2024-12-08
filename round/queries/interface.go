// round/queries/interface.go
package roundqueries

import (
	"context"

	apimodels "github.com/Black-And-White-Club/tcr-bot/round/models"
)

// RoundQueryService defines the interface for querying round data.
type QueryService interface {
	GetRounds(ctx context.Context) ([]*apimodels.Round, error)
	HasActiveRounds(ctx context.Context) (bool, error)
	// ... add other query methods as needed
}
