package userhandlers

import (
	"log/slog"

	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
	"go.opentelemetry.io/otel/trace"
)

// UserHandlers implements the Handlers interface for user events.
type UserHandlers struct {
	service userservice.Service
	helpers utils.Helpers
}

// NewUserHandlers creates a new UserHandlers instance.
// Changed return type to Handlers interface to match guild module pattern.
func NewUserHandlers(
	service userservice.Service,
	logger *slog.Logger,
	tracer trace.Tracer,
	helpers utils.Helpers,
	metrics usermetrics.UserMetrics,
) Handlers {
	return &UserHandlers{
		service: service,
		helpers: helpers,
	}
}

// mapOperationResult manually maps the generic OperationResult to handlerwrapper.Result.
// Updated to use generics [S any, F any] and manual slice construction.
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
