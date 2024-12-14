// In app/modules/round/services/interface.go

package roundservice

import (
	"context"
)

// RoundService defines the interface for round-related services.
type Service interface {
	IsRoundUpcoming(ctx context.Context, roundID int64) (bool, error)
	// ... other service methods as needed ...
}
