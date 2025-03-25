package leaderboardrouter

import (
	"context"
	"fmt"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	leaderboardhandlers "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/handlers"
	"github.com/Black-And-White-Club/frolf-bot/config"
	"github.com/ThreeDotsLabs/watermill/components/metrics"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/prometheus/client_golang/prometheus"
)

// LeaderboardRouter handles routing for leaderboard module events.
type LeaderboardRouter struct {
	logger           observability.Logger
	Router           *message.Router
	subscriber       eventbus.EventBus
	publisher        eventbus.EventBus
	config           *config.Config
	helper           utils.Helpers
	tracer           observability.Tracer
	middlewareHelper utils.MiddlewareHelpers
}

// NewLeaderboardRouter creates a new LeaderboardRouter.
func NewLeaderboardRouter(
	logger observability.Logger,
	router *message.Router,
	subscriber eventbus.EventBus,
	publisher eventbus.EventBus,
	config *config.Config,
	helper utils.Helpers,
	tracer observability.Tracer,
) *LeaderboardRouter {
	return &LeaderboardRouter{
		logger:           logger,
		Router:           router,
		subscriber:       subscriber,
		publisher:        publisher,
		config:           config,
		helper:           helper,
		tracer:           tracer,
		middlewareHelper: utils.NewMiddlewareHelper(),
	}
}

// Configure sets up the router with the necessary handlers and dependencies.
func (r *LeaderboardRouter) Configure(leaderboardService leaderboardservice.Service, eventbus eventbus.EventBus) error {
	// Create Prometheus metrics builder
	metricsBuilder := metrics.NewPrometheusMetricsBuilder(prometheus.NewRegistry(), "", "")
	// Add metrics middleware to the router
	metricsBuilder.AddPrometheusRouterMetrics(r.Router)

	// Create leaderboard handlers with logger and tracer
	leaderboardHandlers := leaderboardhandlers.NewLeaderboardHandlers(leaderboardService, r.logger, r.tracer, r.helper)

	// Add middleware specific to the leaderboard module
	r.Router.AddMiddleware(
		middleware.CorrelationID,
		r.middlewareHelper.CommonMetadataMiddleware("leaderboard"),
		r.middlewareHelper.DiscordMetadataMiddleware(),
		r.middlewareHelper.RoutingMetadataMiddleware(),
		middleware.Recoverer,
		middleware.Retry{MaxRetries: 3}.Middleware,
	)

	if err := r.RegisterHandlers(context.Background(), leaderboardHandlers); err != nil {
		return fmt.Errorf("failed to register handlers: %w", err)
	}
	return nil
}

// RegisterHandlers registers event handlers.
func (r *LeaderboardRouter) RegisterHandlers(ctx context.Context, handlers leaderboardhandlers.Handlers) error {
	r.logger.Info("🚀 Entering RegisterHandlers for Leaderboard")

	eventsToHandlers := map[string]message.HandlerFunc{
		leaderboardevents.RoundFinalized:                    handlers.HandleRoundFinalized,
		leaderboardevents.LeaderboardUpdateRequested:        handlers.HandleLeaderboardUpdateRequested,
		leaderboardevents.TagSwapRequested:                  handlers.HandleTagSwapRequested,
		leaderboardevents.TagSwapInitiated:                  handlers.HandleTagSwapInitiated,
		leaderboardevents.GetLeaderboardRequest:             handlers.HandleGetLeaderboardRequest,
		leaderboardevents.GetTagByUserIDRequest:             handlers.HandleGetTagByUserIDRequest,
		leaderboardevents.TagAssigned:                       handlers.HandleTagAssigned,
		leaderboardevents.TagAvailabilityCheckRequest:       handlers.HandleTagAvailabilityCheckRequested,
		leaderboardevents.LeaderboardTagAssignmentRequested: handlers.HandleTagAssignmentRequested,
	}

	for topic, handlerFunc := range eventsToHandlers {
		handlerName := fmt.Sprintf("leaderboard.%s", topic)

		r.Router.AddHandler(
			handlerName,
			topic,
			r.subscriber,
			"",  // No direct publish topic
			nil, // No manual publisher
			func(msg *message.Message) ([]*message.Message, error) {
				messages, err := handlerFunc(msg)
				if err != nil {
					r.logger.Error("Error processing message", attr.String("message_id", msg.UUID), attr.Any("error", err))
					return nil, err
				}
				for _, m := range messages {
					publishTopic := m.Metadata.Get("topic")
					if publishTopic != "" {
						r.logger.Info("🚀 Auto-publishing message", attr.String("message_id", m.UUID), attr.String("topic", publishTopic))
						if err := r.publisher.Publish(publishTopic, m); err != nil {
							return nil, fmt.Errorf("failed to publish to %s: %w", publishTopic, err)
						}
					} else {
						r.logger.Warn("⚠️ Message missing topic metadata, dropping", attr.String("message_id", m.UUID))
					}
				}
				return nil, nil
			},
		)
	}
	return nil
}

// Close stops the router.
func (r *LeaderboardRouter) Close() error {
	return r.Router.Close()
}
