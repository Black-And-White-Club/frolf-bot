// round/query_service.go
package roundqueries

import (
	"context"
	"fmt"
	"time"

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

// GetScoreForParticipant retrieves the score for a specific participant in a round.
func (s *RoundQueryService) GetScoreForParticipant(ctx context.Context, roundID int64, participantID string) (*rounddb.Score, error) {
	score, err := s.roundDB.GetScoreForParticipant(ctx, roundID, participantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get score for participant: %w", err)
	}
	return score, nil
}

// HasActiveRounds checks if there are any active rounds.
func (s *RoundQueryService) HasActiveRounds(ctx context.Context) (bool, error) {
	// 1. Check for upcoming rounds within the next hour
	now := time.Now()
	oneHourFromNow := now.Add(time.Hour)
	upcomingRounds, err := s.roundDB.GetUpcomingRounds(ctx, now, oneHourFromNow)
	if err != nil {
		return false, fmt.Errorf("failed to get upcoming rounds: %w", err)
	}
	if len(upcomingRounds) > 0 {
		return true, nil // There are upcoming rounds
	}

	// 2. If no upcoming rounds, check for rounds in progress
	rounds, err := s.roundDB.GetRounds(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get rounds: %w", err)
	}
	for _, round := range rounds {
		if round.State == rounddb.RoundStateInProgress {
			return true, nil // There's a round in progress
		}
	}

	// 3. No active rounds found
	return false, nil
}
