package scorehandlers

import (
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	scoreservice "github.com/Black-And-White-Club/frolf-bot/app/modules/score/application"
)

// ScoreHandlers implements the Handlers interface for score events.
type ScoreHandlers struct {
	service scoreservice.Service
	helpers utils.Helpers
}

// NewScoreHandlers creates a new ScoreHandlers instance.
func NewScoreHandlers(
	service scoreservice.Service,
	// logger and tracer parameters are kept for compatibility but ignored by handlers
	_ interface{},
	_ interface{},
	helpers utils.Helpers,
	_ interface{},
) Handlers {
	return &ScoreHandlers{
		service: service,
		helpers: helpers,
	}
}

// mapOperationResult converts a service OperationResult to handler Results.
func mapOperationResult(
	result results.OperationResult,
	successTopic, failureTopic string,
) []handlerwrapper.Result {
	handlerResults := result.MapToHandlerResults(successTopic, failureTopic)

	wrapperResults := make([]handlerwrapper.Result, len(handlerResults))
	for i, hr := range handlerResults {
		wrapperResults[i] = handlerwrapper.Result{
			Topic:    hr.Topic,
			Payload:  hr.Payload,
			Metadata: hr.Metadata,
		}
	}

	return wrapperResults
}
