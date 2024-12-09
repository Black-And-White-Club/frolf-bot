// In round/helpers/interface.go (new file)

package roundhelper

import (
	"context"

	roundconverter "github.com/Black-And-White-Club/tcr-bot/round/converter"
	rounddb "github.com/Black-And-White-Club/tcr-bot/round/db"
	apimodels "github.com/Black-And-White-Club/tcr-bot/round/models"
)

type RoundHelper interface {
	GetRound(ctx context.Context, roundDB rounddb.RoundDB, converter roundconverter.RoundConverter, roundID int64) (*apimodels.Round, error)
	// ... add other helper methods as needed
}
