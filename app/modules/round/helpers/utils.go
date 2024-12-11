// round/helpers/utils.go

package roundhelper

import (
	"context"
	"fmt"

	roundconverter "github.com/Black-And-White-Club/tcr-bot/round/converter"
	rounddb "github.com/Black-And-White-Club/tcr-bot/round/db"
	apimodels "github.com/Black-And-White-Club/tcr-bot/round/models"
)

// ... other imports

// RoundHelperImpl is the concrete implementation of the RoundHelper interface.
type RoundHelperImpl struct {
	Converter roundconverter.RoundConverter // Change to Converter (uppercase C)
}

// NewRoundHelperImpl creates a new RoundHelperImpl.
func NewRoundHelperImpl(converter roundconverter.RoundConverter) *RoundHelperImpl { // Inject converter
	return &RoundHelperImpl{
		Converter: converter,
	}
}

// GetRound retrieves a specific round by ID.
func (rh *RoundHelperImpl) GetRound(ctx context.Context, roundDB rounddb.RoundDB, converter roundconverter.RoundConverter, roundID int64) (*apimodels.Round, error) {
	modelRound, err := roundDB.GetRound(ctx, roundID)
	if err != nil {
		return nil, fmt.Errorf("failed to get round: %w", err)
	}
	return rh.Converter.ConvertModelRoundToStructRound(modelRound), nil // Use rh.converter
}
