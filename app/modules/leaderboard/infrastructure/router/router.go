package leaderboardrouter

import (
	"context"
	"log/slog"
	"os"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	tracingfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/tracing"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	leaderboardhandlers "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/handlers"
	"github.com/Black-And-White-Club/frolf-bot/config"
	"github.com/ThreeDotsLabs/watermill/components/metrics"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
)

const (
	TestEnvironmentFlag  = "APP_ENV"
	TestEnvironmentValue = "test"
)

type LeaderboardRouter struct {
	logger             *slog.Logger
	Router             *message.Router
	subscriber         eventbus.EventBus
	publisher          eventbus.EventBus
	config             *config.Config
	helper             utils.Helpers
	tracer             trace.Tracer
	middlewareHelper   utils.MiddlewareHelpers
	metricsBuilder     *metrics.PrometheusMetricsBuilder
	prometheusRegistry *prometheus.Registry
	metricsEnabled     bool
}

// NewLeaderboardRouter creates a new instance of the router.
// Note: sagaCoordinator is typically set during Configure to allow for
// flexible initialization order in the Module.
func NewLeaderboardRouter(
	logger *slog.Logger,
	router *message.Router,
	subscriber eventbus.EventBus,
	publisher eventbus.EventBus,
	config *config.Config,
	helper utils.Helpers,
	tracer trace.Tracer,
	prometheusRegistry *prometheus.Registry,
) *LeaderboardRouter {
	actualAppEnv := os.Getenv(TestEnvironmentFlag)
	inTestEnv := actualAppEnv == TestEnvironmentValue

	var metricsBuilder *metrics.PrometheusMetricsBuilder
	if prometheusRegistry != nil && !inTestEnv {
		builder := metrics.NewPrometheusMetricsBuilder(prometheusRegistry, "", "")
		metricsBuilder = &builder
	}

	metricsEnabled := prometheusRegistry != nil && !inTestEnv

	return &LeaderboardRouter{
		logger:             logger,
		Router:             router,
		subscriber:         subscriber,
		publisher:          publisher,
		config:             config,
		helper:             helper,
		tracer:             tracer,
		middlewareHelper:   utils.NewMiddlewareHelper(),
		metricsBuilder:     metricsBuilder,
		prometheusRegistry: prometheusRegistry,
		metricsEnabled:     metricsEnabled,
	}
}

// Configure sets up the middlewares and registers all module-specific event handlers.
func (r *LeaderboardRouter) Configure(
	routerCtx context.Context,
	handlers leaderboardhandlers.Handlers,
) error {
	if r.metricsEnabled && r.metricsBuilder != nil {
		r.logger.Info("Adding Prometheus router metrics middleware for Leaderboard")
		r.metricsBuilder.AddPrometheusRouterMetrics(r.Router)
	}

	r.Router.AddMiddleware(
		middleware.CorrelationID,
		r.middlewareHelper.CommonMetadataMiddleware("leaderboard"),
		r.middlewareHelper.DiscordMetadataMiddleware(),
		r.middlewareHelper.RoutingMetadataMiddleware(),
		middleware.Recoverer,
		tracingfrolfbot.TraceHandler(r.tracer),
	)

	return r.RegisterHandlers(routerCtx, handlers)
}

// handlerDeps provides a scannable structure for the registerHandler helper.
type handlerDeps struct {
	router     *message.Router
	subscriber eventbus.EventBus
	publisher  eventbus.EventBus
	logger     *slog.Logger
	tracer     trace.Tracer
	helper     utils.Helpers
	metrics    handlerwrapper.ReturningMetrics
}

// registerHandler is a generic helper to reduce boilerplate when adding topics to the router.
func registerHandler[T any](
	deps handlerDeps,
	topic string,
	handler func(context.Context, *T) ([]handlerwrapper.Result, error),
) {
	handlerName := "leaderboard." + topic
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
			deps.metrics,
			handler,
		),
	)
}

// RegisterHandlers binds specific event topics to their corresponding handler logic.
func (r *LeaderboardRouter) RegisterHandlers(ctx context.Context, handlers leaderboardhandlers.Handlers) error {
	r.logger.InfoContext(ctx, "Registering Leaderboard Event Handlers")

	deps := handlerDeps{
		router:     r.Router,
		subscriber: r.subscriber,
		publisher:  r.publisher,
		logger:     r.logger,
		tracer:     r.tracer,
		helper:     r.helper,
		metrics:    nil, // Handlers return slices of results handled by the wrapper
	}

	// MUTATIONS: Handlers that change leaderboard state and trigger the Traffic Cop/Saga
	registerHandler(deps, leaderboardevents.LeaderboardUpdateRequestedV1, handlers.HandleLeaderboardUpdateRequested)
	registerHandler(deps, leaderboardevents.TagSwapRequestedV1, handlers.HandleTagSwapRequested)
	registerHandler(deps, sharedevents.LeaderboardBatchTagAssignmentRequestedV1, handlers.HandleBatchTagAssignmentRequested)

	// READS: Handlers that query the current state without mutation
	registerHandler(deps, leaderboardevents.GetLeaderboardRequestedV1, handlers.HandleGetLeaderboardRequest)
	registerHandler(deps, sharedevents.DiscordTagLookupRequestedV1, handlers.HandleGetTagByUserIDRequest)
	registerHandler(deps, sharedevents.RoundTagLookupRequestedV1, handlers.HandleRoundGetTagRequest)
	registerHandler(deps, sharedevents.TagAvailabilityCheckRequestedV1, handlers.HandleTagAvailabilityCheckRequested)

	// INFRASTRUCTURE: Handlers managing module lifecycle or configuration
	registerHandler(deps, guildevents.GuildConfigCreatedV1, handlers.HandleGuildConfigCreated)

	return nil
}

// Close stops the router and cleans up resources.
func (r *LeaderboardRouter) Close() error {
	return r.Router.Close()
}
