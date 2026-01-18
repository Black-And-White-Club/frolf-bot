package guildhandlers

import (
	"log/slog"

	guildmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/guild"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	guildservice "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/application"
	"go.opentelemetry.io/otel/trace"
)

// GuildHandlers implements the Handlers interface for guild events.
type GuildHandlers struct {
	service guildservice.Service
	helpers utils.Helpers
}

// NewGuildHandlers creates a new GuildHandlers instance.
func NewGuildHandlers(service guildservice.Service, logger *slog.Logger, tracer trace.Tracer, helpers utils.Helpers, metrics guildmetrics.GuildMetrics) *GuildHandlers {
	return &GuildHandlers{
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
