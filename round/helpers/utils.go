// round/helpers/utils.go

package roundhelper

import (
	"context"
	"fmt"

	converter "github.com/Black-And-White-Club/tcr-bot/round/converter"
	rounddb "github.com/Black-And-White-Club/tcr-bot/round/db" // Importing the db package
	apimodels "github.com/Black-And-White-Club/tcr-bot/round/models"
)

// getRound retrieves a specific round by ID.
func GetRound(ctx context.Context, roundDB rounddb.RoundDB, converter converter.DefaultRoundConverter, roundID int64) (*apimodels.Round, error) {
	modelRound, err := roundDB.GetRound(ctx, roundID)
	if err != nil {
		return nil, fmt.Errorf("failed to get round: %w", err)
	}
	return converter.ConvertModelRoundToStructRound(modelRound), nil
}
