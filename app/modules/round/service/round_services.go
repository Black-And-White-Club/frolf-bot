// In app/modules/round/services/round_service.go

package roundservice

import (
	"context"
	"fmt"

	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
)

type RoundService struct {
	roundDB rounddb.RoundDB
}

func NewRoundService(roundDB rounddb.RoundDB) *RoundService {
	return &RoundService{roundDB: roundDB}
}

func (s *RoundService) IsRoundUpcoming(ctx context.Context, roundID int64) (bool, error) {
	round, err := s.roundDB.GetRound(ctx, roundID)
	if err != nil {
		return false, fmt.Errorf("failed to get round: %w", err)
	}
	return round.State == rounddb.RoundStateUpcoming, nil
}
