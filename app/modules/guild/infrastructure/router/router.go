package guildrouter

import (
	"context"
	"log/slog"
	"os"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	guildhandlers "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/infrastructure/handlers"
	"github.com/Black-And-White-Club/frolf-bot/config"
	"github.com/ThreeDotsLabs/watermill/components/metrics"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
)

const (
	TestEnvironmentFlag  = "APP_ENV"
	TestEnvironmentValue = "test"
)

// GuildRouter handles Watermill handler registration for guild events.
type GuildRouter struct {
	logger             *slog.Logger
	Router             *message.Router
	subscriber         eventbus.EventBus
	publisher          eventbus.EventBus
	config             *config.Config
	helper             utils.Helpers
	tracer             trace.Tracer
	metricsBuilder     *metrics.PrometheusMetricsBuilder
	prometheusRegistry *prometheus.Registry
	metricsEnabled     bool
}

// NewGuildRouter creates a new GuildRouter.
func NewGuildRouter(
	logger *slog.Logger,
	router *message.Router,
	subscriber eventbus.EventBus,
	publisher eventbus.EventBus,
	cfg *config.Config,
	helper utils.Helpers,
	tracer trace.Tracer,
	prometheusRegistry *prometheus.Registry,
) *GuildRouter {
	actualAppEnv := os.Getenv(TestEnvironmentFlag)
	inTestEnv := actualAppEnv == TestEnvironmentValue

	var metricsBuilder *metrics.PrometheusMetricsBuilder
	if prometheusRegistry != nil && !inTestEnv {
		builder := metrics.NewPrometheusMetricsBuilder(prometheusRegistry, "", "")
		metricsBuilder = &builder
	}

	return &GuildRouter{
		logger:             logger,
		Router:             router,
		subscriber:         subscriber,
		publisher:          publisher,
		config:             cfg,
		helper:             helper,
		tracer:             tracer,
		metricsBuilder:     metricsBuilder,
		prometheusRegistry: prometheusRegistry,
		metricsEnabled:     metricsBuilder != nil,
	}
}

// Configure sets up the router with handlers.
func (r *GuildRouter) Configure(_ context.Context, handlers guildhandlers.Handlers) error {
	// Add metrics middleware conditionally
	if r.metricsEnabled && r.metricsBuilder != nil {
		r.metricsBuilder.AddPrometheusRouterMetrics(r.Router)
	}

	// Register all handlers
	r.registerHandlers(handlers)
	return nil
}

// handlerDeps bundles dependencies for handler registration.
type handlerDeps struct {
	router     *message.Router
	subscriber eventbus.EventBus
	publisher  eventbus.EventBus
	logger     *slog.Logger
	tracer     trace.Tracer
	helper     utils.Helpers
}

// registerHandlers wires NATS topics to handler methods.
func (r *GuildRouter) registerHandlers(handlers guildhandlers.Handlers) {
	deps := handlerDeps{
		router:     r.Router,
		subscriber: r.subscriber,
		publisher:  r.publisher,
		logger:     r.logger,
		tracer:     r.tracer,
		helper:     r.helper,
	}

	registerHandler(deps, guildevents.GuildConfigCreationRequestedV1, handlers.HandleCreateGuildConfig)
	registerHandler(deps, guildevents.GuildConfigRetrievalRequestedV1, handlers.HandleRetrieveGuildConfig)
	registerHandler(deps, guildevents.GuildConfigUpdateRequestedV1, handlers.HandleUpdateGuildConfig)
	registerHandler(deps, guildevents.GuildConfigDeletionRequestedV1, handlers.HandleDeleteGuildConfig)
}

// registerHandler is a generic function for type-safe Watermill handler registration.
func registerHandler[T any](
	deps handlerDeps,
	topic string,
	handler func(context.Context, *T) ([]handlerwrapper.Result, error),
) {
	handlerName := "guild." + topic

	deps.router.AddHandler(
		handlerName,
		topic,
		deps.subscriber,
		"",
		deps.publisher,
		handlerwrapper.WrapTransformingTyped(
			handlerName,
			deps.logger,
			deps.tracer,
			deps.helper,
			nil, // metrics can be passed if needed
			handler,
		),
	)
}

// Close shuts down the router.
func (r *GuildRouter) Close() error {
	return r.Router.Close()
}
