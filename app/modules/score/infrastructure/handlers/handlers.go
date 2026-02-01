package scorehandlers

import (
	"log/slog"

	scoremetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/score"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	scoreservice "github.com/Black-And-White-Club/frolf-bot/app/modules/score/application"
	"go.opentelemetry.io/otel/trace"
)

// ScoreHandlers implements the Handlers interface for score events.
type ScoreHandlers struct {
	service scoreservice.Service
	helpers utils.Helpers
}

// NewScoreHandlers creates a new ScoreHandlers instance.
// Now using concrete types for logger and tracer to match the pattern in guildhandlers.
func NewScoreHandlers(
	service scoreservice.Service,
	logger *slog.Logger,
	tracer trace.Tracer,
	helpers utils.Helpers,
	metrics scoremetrics.ScoreMetrics,
) Handlers {
	return &ScoreHandlers{
		service: service,
		helpers: helpers,
	}
}

// mapOperationResult manually maps the generic OperationResult to handlerwrapper.Result.
// Updated to use [S any, F any] to match the refactored results package.
func mapOperationResult[S any, F any](
	result results.OperationResult[S, F],
	successTopic, failureTopic string,
) []handlerwrapper.Result {
	var wrapperResults []handlerwrapper.Result

	// Handle Success case
	if result.Success != nil {
		wrapperResults = append(wrapperResults, handlerwrapper.Result{
			Topic:   successTopic,
			Payload: result.Success,
		})
	}

	// Handle Failure case
	if result.Failure != nil {
		wrapperResults = append(wrapperResults, handlerwrapper.Result{
			Topic:   failureTopic,
			Payload: result.Failure,
		})
	}

	return wrapperResults
}
