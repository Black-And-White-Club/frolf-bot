// round/query_service.go
package roundqueries

import (
	"context"
	"fmt"

	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
)

// QueryService handles query-related logic for rounds.
type RoundQueryService struct {
	roundDB rounddb.RoundDB
}

// NewRoundQueryService creates a new RoundQueryService.
func NewRoundQueryService(roundDB rounddb.RoundDB) *RoundQueryService {
	return &RoundQueryService{
		roundDB: roundDB,
	}
}

// GetRounds retrieves all rounds.
func (s *RoundQueryService) GetRounds(ctx context.Context) ([]*rounddb.Round, error) {
	rounds, err := s.roundDB.GetRounds(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get rounds: %w", err)
	}
	return rounds, nil
}

// GetRound retrieves a specific round by ID.
func (s *RoundQueryService) GetRound(ctx context.Context, roundID int64) (*rounddb.Round, error) {
	round, err := s.roundDB.GetRound(ctx, roundID)
	if err != nil {
		return nil, fmt.Errorf("failed to get round: %w", err)
	}
	return round, nil
}
