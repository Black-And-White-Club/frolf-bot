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
func NewUserHandlers(service userservice.Service, logger *slog.Logger, tracer trace.Tracer, helpers utils.Helpers, metrics usermetrics.UserMetrics) *UserHandlers {
	return &UserHandlers{
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
