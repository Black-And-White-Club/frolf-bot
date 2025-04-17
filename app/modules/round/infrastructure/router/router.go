package roundrouter

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	tracingfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/tracing"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	roundhandlers "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/handlers"
	"github.com/Black-And-White-Club/frolf-bot/config"
	"github.com/ThreeDotsLabs/watermill/components/metrics"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
)

// RoundRouter handles routing for round module events.
type RoundRouter struct {
	logger           *slog.Logger
	Router           *message.Router
	subscriber       eventbus.EventBus
	publisher        eventbus.EventBus
	config           *config.Config
	helper           utils.Helpers
	tracer           trace.Tracer
	middlewareHelper utils.MiddlewareHelpers
}

// NewRoundRouter creates a new RoundRouter.
func NewRoundRouter(
	logger *slog.Logger,
	router *message.Router,
	subscriber eventbus.EventBus,
	publisher eventbus.EventBus,
	config *config.Config,
	helper utils.Helpers,
	tracer trace.Tracer,
) *RoundRouter {
	return &RoundRouter{
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
func (r *RoundRouter) Configure(roundService roundservice.Service, eventbus eventbus.EventBus, roundMetrics roundmetrics.RoundMetrics) error {
	// Create Prometheus metrics builder
	metricsBuilder := metrics.NewPrometheusMetricsBuilder(prometheus.NewRegistry(), "", "")
	// Add metrics middleware to the router
	metricsBuilder.AddPrometheusRouterMetrics(r.Router)

	// Create round handlers with logger and tracer
	roundHandlers := roundhandlers.NewRoundHandlers(roundService, r.logger, r.tracer, r.helper, roundMetrics)

	// Add middleware specific to the round module
	r.Router.AddMiddleware(
		middleware.CorrelationID,
		r.middlewareHelper.CommonMetadataMiddleware("round"),
		r.middlewareHelper.DiscordMetadataMiddleware(),
		r.middlewareHelper.RoutingMetadataMiddleware(),
		middleware.Recoverer,
		middleware.Retry{MaxRetries: 3}.Middleware,
		tracingfrolfbot.TraceHandler(r.tracer),
	)

	if err := r.RegisterHandlers(context.Background(), roundHandlers); err != nil {
		return fmt.Errorf("failed to register handlers: %w", err)
	}
	return nil
}

// RegisterHandlers registers event handlers.
func (r *RoundRouter) RegisterHandlers(ctx context.Context, handlers roundhandlers.Handlers) error {
	r.logger.InfoContext(ctx, "Entering RegisterHandlers for Round")

	eventsToHandlers := map[string]message.HandlerFunc{
		roundevents.RoundCreateRequest:                    handlers.HandleCreateRoundRequest,
		roundevents.RoundStored:                           handlers.HandleRoundStored,
		roundevents.RoundUpdateRequest:                    handlers.HandleRoundUpdateRequest,
		roundevents.RoundUpdateValidated:                  handlers.HandleRoundUpdateValidated,
		roundevents.RoundFinalized:                        handlers.HandleRoundFinalized,
		roundevents.RoundDeleteRequest:                    handlers.HandleRoundDeleteRequest,
		roundevents.RoundDeleteAuthorized:                 handlers.HandleRoundDeleteAuthorized,
		roundevents.RoundParticipantJoinRequest:           handlers.HandleParticipantJoinRequest,
		roundevents.RoundParticipantJoinValidationRequest: handlers.HandleParticipantJoinValidationRequest,
		roundevents.RoundParticipantRemovalRequest:        handlers.HandleParticipantRemovalRequest,
		roundevents.RoundScoreUpdateRequest:               handlers.HandleScoreUpdateRequest,
		roundevents.RoundScoreUpdateValidated:             handlers.HandleScoreUpdateValidated,
		roundevents.RoundAllScoresSubmitted:               handlers.HandleAllScoresSubmitted,
		roundevents.RoundReminder:                         handlers.HandleRoundReminder,
		roundevents.RoundStarted:                          handlers.HandleRoundStarted,
		roundevents.RoundTagNumberFound:                   handlers.HandleTagNumberFound,
		roundevents.RoundTagNumberNotFound:                handlers.HandleTagNumberNotFound,
	}

	for topic, handlerFunc := range eventsToHandlers {
		handlerName := fmt.Sprintf("round.%s", topic)
		r.Router.AddHandler(
			handlerName,
			topic,
			r.subscriber,
			"",  // No direct publish topic
			nil, // No manual publisher
			func(msg *message.Message) ([]*message.Message, error) {
				messages, err := handlerFunc(msg)
				if err != nil {
					r.logger.ErrorContext(ctx, "Error processing message", attr.String("message_id", msg.UUID), attr.Any("error", err))
					return nil, err
				}
				for _, m := range messages {
					publishTopic := m.Metadata.Get("topic")
					if publishTopic != "" {
						r.logger.InfoContext(ctx, "🚀 Auto-publishing message", attr.String("message_id", m.UUID), attr.String("topic", publishTopic))
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
func (r *RoundRouter) Close() error {
	return r.Router.Close()
}
