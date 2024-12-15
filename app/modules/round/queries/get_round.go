// In app/modules/round/queries/get_round.go

package roundqueries

import (
	"context"
	"fmt"

	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
)

// GetRoundQuery represents the query to get a specific round by ID.
type GetRoundQuery struct {
	RoundID int64 `json:"round_id"`
}

// GetRoundHandler handles the GetRoundQuery.
type GetRoundHandler interface {
	Handle(ctx context.Context, query GetRoundQuery) (*rounddb.Round, error)
}

type getRoundHandler struct {
	roundDB rounddb.RoundDB
}

// NewGetRoundHandler creates a new getRoundHandler.
func NewGetRoundHandler(roundDB rounddb.RoundDB) *getRoundHandler {
	return &getRoundHandler{roundDB: roundDB}
}

// Handle processes the GetRoundQuery.
func (h *getRoundHandler) Handle(ctx context.Context, query GetRoundQuery) (*rounddb.Round, error) {
	round, err := h.roundDB.GetRound(ctx, query.RoundID)
	if err != nil {
		return nil, fmt.Errorf("failed to get round: %w", err)
	}
	return round, nil
}
