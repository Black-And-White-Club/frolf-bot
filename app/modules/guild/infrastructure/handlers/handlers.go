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
func NewGuildHandlers(
	service guildservice.Service,
	logger *slog.Logger,
	tracer trace.Tracer,
	helpers utils.Helpers,
	metrics guildmetrics.GuildMetrics,
) Handlers {
	return &GuildHandlers{
		service: service,
		helpers: helpers,
	}
}

// mapOperationResult manually maps the generic OperationResult to handlerwrapper.Result.
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
