package leaderboardrouter

import (
	"context"
	"log/slog"
	"os"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	leaderboardhandlers "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/handlers"
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

// registerFanOutHandler is a specialized helper for handlers that return multiple messages
// intended for different topics. It intercepts the results and publishes them individually
// via the EventBus to ensure they land on the correct NATS subjects.
func registerFanOutHandler[T any](
	deps handlerDeps,
	topic string,
	handler func(context.Context, *T) ([]handlerwrapper.Result, error),
) {
	handlerName := "leaderboard." + topic

	// 1. Get the standard wrapped handler (handles JSON, tracing, etc.)
	wrappedHandler := handlerwrapper.WrapTransformingTyped(
		handlerName,
		deps.logger,
		deps.tracer,
		deps.helper,
		deps.metrics,
		handler,
	)

	// 2. Create the Dispatcher Interceptor
	dispatcher := func(msg *message.Message) ([]*message.Message, error) {
		// Run the business logic
		results, err := wrappedHandler(msg)
		if err != nil {
			return nil, err
		}

		// 3. Loop through every result and publish to its specific topic
		for _, outMsg := range results {
			// Extract the topic from the metadata set by mapSuccessResults
			targetTopic := outMsg.Metadata.Get("topic")
			if targetTopic == "" {
				targetTopic = outMsg.Metadata.Get("Topic")
			}

			if targetTopic != "" {
				// Use the eventbus to publish so we get all our standard logs/metrics
				if err := deps.publisher.Publish(targetTopic, outMsg); err != nil {
					deps.logger.Error("fan-out publish failed",
						"topic", targetTopic,
						"error", err,
					)
					return nil, err
				}
			} else {
				deps.logger.Warn("message skipped: no topic metadata found", "message_id", outMsg.UUID)
			}
		}

		// 4. Return nil so the Watermill Router doesn't try to publish anything
		return nil, nil
	}

	// Register the dispatcher to the router
	deps.router.AddHandler(
		handlerName,
		topic,
		deps.subscriber,
		"", // Default output topic is ignored
		deps.publisher,
		dispatcher,
	)
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
		metrics:    nil,
	}

	// MUTATIONS: Handlers that change leaderboard state
	registerHandler(deps, leaderboardevents.LeaderboardUpdateRequestedV1, handlers.HandleLeaderboardUpdateRequested)
	registerHandler(deps, leaderboardevents.TagSwapRequestedV1, handlers.HandleTagSwapRequested)

	// SPECIAL CASE: Use registerFanOutHandler for Batch Tag Assignments
	// because it triggers both Leaderboard events and Round sync events.
	registerFanOutHandler(deps, sharedevents.LeaderboardBatchTagAssignmentRequestedV1, handlers.HandleBatchTagAssignmentRequested)

	// READS: Handlers that query the current state
	registerHandler(deps, leaderboardevents.GetLeaderboardRequestedV1, handlers.HandleGetLeaderboardRequest)
	registerHandler(deps, "leaderboard.snapshot.request.v1.>", handlers.HandleGetLeaderboardRequest)
	registerHandler(deps, sharedevents.DiscordTagLookupRequestedV1, handlers.HandleGetTagByUserIDRequest)
	registerHandler(deps, sharedevents.RoundTagLookupRequestedV1, handlers.HandleRoundGetTagRequest)
	registerHandler(deps, sharedevents.TagAvailabilityCheckRequestedV1, handlers.HandleTagAvailabilityCheckRequested)

	// ADMIN OPERATIONS
	registerHandler(deps, leaderboardevents.LeaderboardPointHistoryRequestedV1, handlers.HandlePointHistoryRequested)
	registerHandler(deps, leaderboardevents.LeaderboardManualPointAdjustmentV1, handlers.HandleManualPointAdjustment)
	registerHandler(deps, leaderboardevents.LeaderboardRecalculateRoundV1, handlers.HandleRecalculateRound)
	registerHandler(deps, leaderboardevents.LeaderboardStartNewSeasonV1, handlers.HandleStartNewSeason)
	registerHandler(deps, leaderboardevents.LeaderboardGetSeasonStandingsV1, handlers.HandleGetSeasonStandings)

	// INFRASTRUCTURE
	registerHandler(deps, guildevents.GuildConfigCreatedV1, handlers.HandleGuildConfigCreated)

	return nil
}

// Close stops the router and cleans up resources.
func (r *LeaderboardRouter) Close() error {
	return r.Router.Close()
}
