// round/queries/interface.go
package roundqueries

import (
	"context"

	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
)

// QueryService defines the interface for querying round data.
type QueryService interface {
	GetRounds(ctx context.Context) ([]*rounddb.Round, error)
	GetRound(ctx context.Context, roundID int64) (*rounddb.Round, error)
	GetScoreForParticipant(ctx context.Context, roundID int64, participantID string) (*rounddb.Score, error) // Add this method
	HasActiveRounds(ctx context.Context) (bool, error)
	// ... add other query methods as needed
}
