package adapters

import (
	"context"
	"fmt"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
)

// RoundLookupAdapter adapts the round service to the leaderboard handler's RoundLookup port.
type RoundLookupAdapter struct {
	roundService roundservice.Service
}

func NewRoundLookupAdapter(roundService roundservice.Service) *RoundLookupAdapter {
	return &RoundLookupAdapter{roundService: roundService}
}

func (a *RoundLookupAdapter) GetRound(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) (*roundtypes.Round, error) {
	result, err := a.roundService.GetRound(ctx, guildID, roundID)
	if err != nil {
		return nil, err
	}
	if result.IsFailure() {
		if result.Failure == nil {
			return nil, fmt.Errorf("round lookup failed with nil error")
		}
		return nil, fmt.Errorf("round lookup failed: %w", *result.Failure)
	}
	// If for some reason Success is nil (and no failure), we return nil, nil
	if result.Success == nil {
		return nil, nil
	}
	return *result.Success, nil
}
