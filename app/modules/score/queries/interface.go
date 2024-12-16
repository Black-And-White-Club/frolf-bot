package scorequeries

import (
	"context"

	scoredb "github.com/Black-And-White-Club/tcr-bot/app/modules/score/db"
)

// ScoreQueryService defines the methods for querying score data.
type QueryService interface {
	GetScore(ctx context.Context, query *GetScoreQuery) (*scoredb.Score, error)
	// Add other query methods as needed
}
