// round/query_service.go
package roundqueries

import (
	"context"
	"fmt"
	"time"

	roundconverter "github.com/Black-And-White-Club/tcr-bot/round/converter"
	rounddb "github.com/Black-And-White-Club/tcr-bot/round/db"
	roundhelper "github.com/Black-And-White-Club/tcr-bot/round/helpers"
	apimodels "github.com/Black-And-White-Club/tcr-bot/round/models"
)

// RoundQueryService handles query-related logic for rounds.
type RoundQueryService struct {
	roundDB   rounddb.RoundDB
	converter roundconverter.RoundConverter // Use the RoundConverter interface
	helper    roundhelper.RoundHelper       // Add the RoundHelper field
}

// NewRoundQueryService creates a new RoundQueryService.
func NewRoundQueryService(roundDB rounddb.RoundDB, converter roundconverter.RoundConverter) QueryService { // Inject converter
	return &RoundQueryService{
		roundDB:   roundDB,
		converter: converter,                                          // Assign the injected converter
		helper:    &roundhelper.RoundHelperImpl{Converter: converter}, // Inject converter into the helper
	}
}

// GetRounds retrieves all rounds.
func (s *RoundQueryService) GetRounds(ctx context.Context) ([]*apimodels.Round, error) {
	modelRounds, err := s.roundDB.GetRounds(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get rounds: %w", err)
	}

	var apiRounds []*apimodels.Round
	for _, modelRound := range modelRounds {
		apiRounds = append(apiRounds, s.converter.ConvertModelRoundToStructRound(modelRound))
	}

	return apiRounds, nil
}

// GetRound retrieves a specific round by ID.
func (s *RoundQueryService) GetRound(ctx context.Context, roundID int64) (*apimodels.Round, error) {
	return s.helper.GetRound(ctx, s.roundDB, s.converter, roundID) // Pass the converter
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
	rounds, err := s.GetRounds(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get rounds: %w", err)
	}
	for _, round := range rounds {
		if round.State == apimodels.RoundStateInProgress {
			return true, nil // There's a round in progress
		}
	}

	// 3. No active rounds found
	return false, nil
}
