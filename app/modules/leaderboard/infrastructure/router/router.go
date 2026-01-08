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
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
	tracingfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/tracing"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
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

// RegisterHandlers registers event handlers using V1 versioned event constants.
func (r *LeaderboardRouter) RegisterHandlers(ctx context.Context, handlers leaderboardhandlers.Handlers) error {
	r.logger.InfoContext(ctx, "Entering Register Handlers for Leaderboard")
	eventsToHandlers := map[string]message.HandlerFunc{
		// Leaderboard Update Flow (from updates.go)
		leaderboardevents.LeaderboardUpdateRequestedV1: handlers.HandleLeaderboardUpdateRequested,

		// Tag Swap Flow (from tags.go)
		leaderboardevents.TagSwapRequestedV1: handlers.HandleTagSwapRequested,

		// Leaderboard Retrieval Flow (from updates.go)
		leaderboardevents.GetLeaderboardRequestedV1: handlers.HandleGetLeaderboardRequest,

		// Discord Tag Lookup Flow (from shared/tags.go)
		sharedevents.DiscordTagLookupRequestedV1: handlers.HandleGetTagByUserIDRequest,

		// Tag Availability Check Flow (from tags.go)
		leaderboardevents.TagAvailabilityCheckRequestedV1: handlers.HandleTagAvailabilityCheckRequested,

		// Batch Tag Assignment Flow (from shared/tags.go)
		sharedevents.LeaderboardBatchTagAssignmentRequestedV1: handlers.HandleBatchTagAssignmentRequested,

		// Round Tag Lookup Flow (from tags.go)
		leaderboardevents.RoundGetTagByUserIDRequestedV1: handlers.HandleRoundGetTagRequest,

		// Cross-module: Guild Config Created (from guild/config.go)
		guildevents.GuildConfigCreatedV1: handlers.HandleGuildConfigCreated,
	}

	for topic, handlerFunc := range eventsToHandlers {
		handlerName := fmt.Sprintf("leaderboard.%s", topic)
		r.Router.AddHandler(
			handlerName,
			topic,
			r.subscriber,
			"",
			nil,
			func(msg *message.Message) ([]*message.Message, error) {
				messages, err := handlerFunc(msg)
				if err != nil {
					r.logger.ErrorContext(ctx, "Error processing message by handler", attr.String("message_id", msg.UUID), attr.Any("error", err))
					return nil, err
				}

				for _, m := range messages {
					publishTopic := r.getPublishTopic(handlerName, m)

					if err := r.publisher.Publish(publishTopic, m); err != nil {
						r.logger.ErrorContext(ctx, "Failed to publish message from handler return", attr.String("message_id", m.UUID), attr.String("topic", publishTopic), attr.Error(err))
						return nil, fmt.Errorf("failed to publish to %s: %w", publishTopic, err)
					}
				}

				return nil, nil
			},
		)
	}
	return nil
}

// getPublishTopic resolves the topic to publish for a given handler's returned message.
// During the migration we keep the metadata fallback for handlers that can emit
// multiple different outcome events. As handlers are stabilized we can make
// explicit mappings here and remove the metadata fallback.
func (r *LeaderboardRouter) getPublishTopic(handlerName string, msg *message.Message) string {
	switch handlerName {
	// No deterministic single-output handlers in this router yet. Keep metadata
	// fallback so multi-outcome handlers continue to work during migration.
	default:
		return msg.Metadata.Get("topic")
	}
}

// Close stops the router.
func (r *LeaderboardRouter) Close() error {
	return r.Router.Close()
}
