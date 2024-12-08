// round/helpers.go
package round

import (
	"context"
	"fmt"

	rounddb "github.com/Black-And-White-Club/tcr-bot/round/db"
)

// getRound retrieves a specific round by ID.
func getRound(ctx context.Context, roundDB rounddb.RoundDB, converter RoundConverter, roundID int64) (*Round, error) {
	modelRound, err := roundDB.GetRound(ctx, roundID)
	if err != nil {
		return nil, fmt.Errorf("failed to get round: %w", err)
	}
	return converter.ConvertModelRoundToStructRound(modelRound), nil
}
