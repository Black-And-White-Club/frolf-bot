package scorehandlers

import (
	"context"

	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// Handlers interface defines the methods that a set of score handlers should implement.
type Handlers interface {
	HandleProcessRoundScoresRequest(ctx context.Context, payload *sharedevents.ProcessRoundScoresRequestedPayloadV1) ([]handlerwrapper.Result, error)
	HandleCorrectScoreRequest(ctx context.Context, payload *sharedevents.ScoreUpdateRequestedPayloadV1) ([]handlerwrapper.Result, error)
	HandleBulkCorrectScoreRequest(ctx context.Context, payload *sharedevents.ScoreBulkUpdateRequestedPayloadV1) ([]handlerwrapper.Result, error)
	HandleReprocessAfterScoreUpdate(ctx context.Context, payload interface{}) ([]handlerwrapper.Result, error)
}
