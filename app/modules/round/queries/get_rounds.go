// In app/modules/round/queries/get_rounds.go

package roundqueries

import (
	"context"
	"fmt"

	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
)

// GetRoundsQuery represents the query to get all rounds.
type GetRoundsQuery struct{}

// GetRoundsHandler handles the GetRoundsQuery.
type GetRoundsHandler interface {
	Handle(ctx context.Context, query GetRoundsQuery) ([]*rounddb.Round, error)
}

type getRoundsHandler struct {
	roundDB rounddb.RoundDB
}

// NewGetRoundsHandler creates a new getRoundsHandler.
func NewGetRoundsHandler(roundDB rounddb.RoundDB) *getRoundsHandler {
	return &getRoundsHandler{roundDB: roundDB}
}

// Handle processes the GetRoundsQuery.
func (h *getRoundsHandler) Handle(ctx context.Context, query GetRoundsQuery) ([]*rounddb.Round, error) {
	rounds, err := h.roundDB.GetRounds(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get rounds: %w", err)
	}
	return rounds, nil
}
