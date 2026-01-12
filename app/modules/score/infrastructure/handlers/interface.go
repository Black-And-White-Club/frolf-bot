package scorehandlers

import (
	"context"

	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// Handlers interface defines the methods that a set of score handlers should implement.
type Handlers interface {
	HandleProcessRoundScoresRequest(ctx context.Context, payload *scoreevents.ProcessRoundScoresRequestedPayloadV1) ([]handlerwrapper.Result, error)
	HandleCorrectScoreRequest(ctx context.Context, payload *scoreevents.ScoreUpdateRequestedPayloadV1) ([]handlerwrapper.Result, error)
	HandleBulkCorrectScoreRequest(ctx context.Context, payload *scoreevents.ScoreBulkUpdateRequestPayload) ([]handlerwrapper.Result, error)
	HandleReprocessAfterScoreUpdate(ctx context.Context, payload interface{}) ([]handlerwrapper.Result, error)
}
