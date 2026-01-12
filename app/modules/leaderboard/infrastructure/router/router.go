package leaderboardrouter

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
	tracingfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/tracing"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	leaderboardhandlers "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/handlers"
	"github.com/Black-And-White-Club/frolf-bot/config"
	"github.com/ThreeDotsLabs/watermill/components/metrics"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
)

// --- Updated Router ---

const (
	TestEnvironmentFlag  = "APP_ENV"
	TestEnvironmentValue = "test"
)

// LeaderboardRouter handles routing for leaderboard module events.
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

// NewLeaderboardRouter creates a new LeaderboardRouter.
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
	// Add logging to check environment variable and conditions
	actualAppEnv := os.Getenv(TestEnvironmentFlag)
	logger.Info("NewLeaderboardRouter: Environment check",
		"APP_ENV_Actual", actualAppEnv,
		"TestEnvironmentValue", TestEnvironmentValue,
		"prometheusRegistryProvided", prometheusRegistry != nil,
	)

	// Check if the application is running in the test environment
	inTestEnv := actualAppEnv == TestEnvironmentValue
	logger.Info("NewLeaderboardRouter: inTestEnv determined", "inTestEnv", inTestEnv)

	var metricsBuilder *metrics.PrometheusMetricsBuilder
	// Only create the metrics builder if a registry is provided AND we are NOT in the test environment
	// Add logging for the condition
	if prometheusRegistry != nil && !inTestEnv {
		logger.Info("NewLeaderboardRouter: Creating Prometheus metrics builder")
		builder := metrics.NewPrometheusMetricsBuilder(prometheusRegistry, "", "")
		metricsBuilder = &builder
	} else {
		logger.Info("NewLeaderboardRouter: Skipping Prometheus metrics builder creation",
			"prometheusRegistryProvided", prometheusRegistry != nil,
			"inTestEnv", inTestEnv,
		)
	}

	// metricsEnabled is true only if a registry is provided AND we are NOT in the test environment
	metricsEnabled := prometheusRegistry != nil && !inTestEnv
	logger.Info("NewLeaderboardRouter: metricsEnabled determined", "metricsEnabled", metricsEnabled)

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
		metricsEnabled:     metricsEnabled, // Use the determined value
	}
}

// Configure sets up the router.
func (r *LeaderboardRouter) Configure(routerCtx context.Context, leaderboardService leaderboardservice.Service, eventbus eventbus.EventBus, leaderboardMetrics leaderboardmetrics.LeaderboardMetrics) error {
	// Add logging before the conditional check for adding middleware
	r.logger.Info("Configure: Checking metricsEnabled before adding middleware",
		"metricsEnabled", r.metricsEnabled,
		"metricsBuilderNil", r.metricsBuilder == nil,
	)

	// Conditionally add Prometheus metrics middleware based on the metricsEnabled flag
	if r.metricsEnabled && r.metricsBuilder != nil {
		r.logger.Info("Adding Prometheus router metrics middleware for Leaderboard")
		r.metricsBuilder.AddPrometheusRouterMetrics(r.Router)
	} else {
		// This log message confirms that metrics are being skipped in the test environment
		r.logger.Info("Skipping Prometheus router metrics middleware for Leaderboard - either in test environment or metrics not configured")
	}

	// Create leaderboard handlers with logger and tracer
	leaderboardHandlers := leaderboardhandlers.NewLeaderboardHandlers(leaderboardService, r.logger, r.tracer, r.helper, leaderboardMetrics)
	// Add middleware specific to the leaderboard module
	r.Router.AddMiddleware(
		middleware.CorrelationID,
		r.middlewareHelper.CommonMetadataMiddleware("leaderboard"),
		r.middlewareHelper.DiscordMetadataMiddleware(),
		r.middlewareHelper.RoutingMetadataMiddleware(),
		middleware.Recoverer,
		tracingfrolfbot.TraceHandler(r.tracer),
	)

	// Pass the routerCtx to RegisterHandlers
	if err := r.RegisterHandlers(routerCtx, leaderboardHandlers); err != nil {
		return fmt.Errorf("failed to register handlers: %w", err)
	}
	return nil
}

type handlerDeps struct {
	router     *message.Router
	subscriber eventbus.EventBus
	publisher  eventbus.EventBus
	logger     *slog.Logger
	tracer     trace.Tracer
	helper     utils.Helpers
	metrics    handlerwrapper.ReturningMetrics
}

// registerHandler registers a pure transformation-pattern handler with typed payload
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
		"", // Watermill reads topic from message metadata when empty
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

// RegisterHandlers registers event handlers using V1 versioned event constants with pure transformation pattern.
func (r *LeaderboardRouter) RegisterHandlers(ctx context.Context, handlers leaderboardhandlers.Handlers) error {
	r.logger.InfoContext(ctx, "Entering Register Handlers for Leaderboard")

	var metrics handlerwrapper.ReturningMetrics // reserved for metrics integration

	deps := handlerDeps{
		router:     r.Router,
		subscriber: r.subscriber,
		publisher:  r.publisher,
		logger:     r.logger,
		tracer:     r.tracer,
		helper:     r.helper,
		metrics:    metrics,
	}

	// Register all leaderboard module handlers using pure transformation pattern
	registerHandler(deps, leaderboardevents.LeaderboardUpdateRequestedV1, handlers.HandleLeaderboardUpdateRequested)
	registerHandler(deps, leaderboardevents.TagSwapRequestedV1, handlers.HandleTagSwapRequested)
	registerHandler(deps, leaderboardevents.GetLeaderboardRequestedV1, handlers.HandleGetLeaderboardRequest)
	registerHandler(deps, sharedevents.DiscordTagLookupRequestedV1, handlers.HandleGetTagByUserIDRequest)
	registerHandler(deps, leaderboardevents.TagAvailabilityCheckRequestedV1, handlers.HandleTagAvailabilityCheckRequested)
	registerHandler(deps, sharedevents.LeaderboardBatchTagAssignmentRequestedV1, handlers.HandleBatchTagAssignmentRequested)
	// Register the V1 shared-events topic for round tag lookup.
	registerHandler(deps, sharedevents.RoundTagLookupRequestedV1, handlers.HandleRoundGetTagRequest)
	registerHandler(deps, guildevents.GuildConfigCreatedV1, handlers.HandleGuildConfigCreated)

	return nil
}

// Close stops the router.
func (r *LeaderboardRouter) Close() error {
	return r.Router.Close()
}
