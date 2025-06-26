package roundrouter

import (
	"context"
	"fmt"
	"log/slog"
	"os" // Import os for environment variable check

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
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

// Constants for environment check.
const (
	TestEnvironmentFlag  = "APP_ENV"
	TestEnvironmentValue = "test"
)

// RoundRouter handles routing for round module events.
type RoundRouter struct {
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

// NewRoundRouter creates a new RoundRouter.
// It now accepts a prometheusRegistry to conditionally enable metrics.
func NewRoundRouter(
	logger *slog.Logger,
	router *message.Router,
	subscriber eventbus.EventBus,
	publisher eventbus.EventBus,
	config *config.Config,
	helper utils.Helpers,
	tracer trace.Tracer,
	prometheusRegistry *prometheus.Registry,
) *RoundRouter {
	// Add logging to check environment variable and conditions
	actualAppEnv := os.Getenv(TestEnvironmentFlag)
	logger.Info("NewRoundRouter: Environment check",
		"APP_ENV_Actual", actualAppEnv,
		"TestEnvironmentValue", TestEnvironmentValue,
		"prometheusRegistryProvided", prometheusRegistry != nil,
	)

	// Check if the application is running in the test environment
	inTestEnv := actualAppEnv == TestEnvironmentValue
	logger.Info("NewRoundRouter: inTestEnv determined", "inTestEnv", inTestEnv)

	var metricsBuilder *metrics.PrometheusMetricsBuilder
	// Only create the metrics builder if a registry is provided AND we are NOT in the test environment
	// Add logging for the condition
	if prometheusRegistry != nil && !inTestEnv {
		logger.Info("NewRoundRouter: Creating Prometheus metrics builder")
		builder := metrics.NewPrometheusMetricsBuilder(prometheusRegistry, "", "")
		metricsBuilder = &builder
	} else {
		logger.Info("NewRoundRouter: Skipping Prometheus metrics builder creation",
			"prometheusRegistryProvided", prometheusRegistry != nil,
			"inTestEnv", inTestEnv,
		)
	}

	// metricsEnabled is true only if a registry is provided AND we are NOT in the test environment
	metricsEnabled := prometheusRegistry != nil && !inTestEnv
	logger.Info("NewRoundRouter: metricsEnabled determined", "metricsEnabled", metricsEnabled)

	return &RoundRouter{
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

// Configure sets up the router with the necessary handlers and dependencies.
func (r *RoundRouter) Configure(routerCtx context.Context, roundService roundservice.Service, eventbus eventbus.EventBus, roundMetrics roundmetrics.RoundMetrics) error {
	r.logger.Info("Configure: Checking metricsEnabled before adding middleware",
		"metricsEnabled", r.metricsEnabled,
		"metricsBuilderNil", r.metricsBuilder == nil,
	)

	// Conditionally add Prometheus metrics middleware based on the metricsEnabled flag
	if r.metricsEnabled && r.metricsBuilder != nil {
		r.logger.Info("Adding Prometheus router metrics middleware for Round")
		r.metricsBuilder.AddPrometheusRouterMetrics(r.Router)
	} else {
		// This log message confirms that metrics are being skipped in the test environment
		r.logger.Info("Skipping Prometheus router metrics middleware for Round - either in test environment or metrics not configured")
	}

	// Create round handlers with logger and tracer
	roundHandlers := roundhandlers.NewRoundHandlers(roundService, r.logger, r.tracer, r.helper, roundMetrics)

	// Add middleware specific to the round module
	r.Router.AddMiddleware(
		middleware.CorrelationID,
		r.middlewareHelper.CommonMetadataMiddleware("round"),
		r.middlewareHelper.DiscordMetadataMiddleware(),
		r.middlewareHelper.RoutingMetadataMiddleware(),
		middleware.Recoverer,
		tracingfrolfbot.TraceHandler(r.tracer),
	)

	// Pass routerCtx to RegisterHandlers
	if err := r.RegisterHandlers(routerCtx, roundHandlers); err != nil {
		return fmt.Errorf("failed to register handlers: %w", err)
	}
	return nil
}

// RegisterHandlers registers event handlers.
func (r *RoundRouter) RegisterHandlers(ctx context.Context, handlers roundhandlers.Handlers) error {
	r.logger.InfoContext(ctx, "Entering RegisterHandlers for Round")

	eventsToHandlers := map[string]message.HandlerFunc{
		roundevents.RoundCreateRequest:                    handlers.HandleCreateRoundRequest,
		roundevents.RoundUpdateRequest:                    handlers.HandleRoundUpdateRequest,
		roundevents.RoundUpdateValidated:                  handlers.HandleRoundUpdateValidated,
		roundevents.RoundFinalized:                        handlers.HandleRoundFinalized,
		roundevents.RoundDeleteRequest:                    handlers.HandleRoundDeleteRequest,
		roundevents.RoundDeleteValidated:                  handlers.HandleRoundDeleteValidated,
		roundevents.RoundDeleteAuthorized:                 handlers.HandleRoundDeleteAuthorized,
		roundevents.RoundParticipantJoinRequest:           handlers.HandleParticipantJoinRequest,
		roundevents.RoundParticipantJoinValidationRequest: handlers.HandleParticipantJoinValidationRequest,
		roundevents.RoundParticipantRemovalRequest:        handlers.HandleParticipantRemovalRequest,
		roundevents.RoundScoreUpdateRequest:               handlers.HandleScoreUpdateRequest,
		roundevents.RoundScoreUpdateValidated:             handlers.HandleScoreUpdateValidated,
		roundevents.RoundAllScoresSubmitted:               handlers.HandleAllScoresSubmitted,
		roundevents.RoundReminder:                         handlers.HandleRoundReminder,
		roundevents.RoundStarted:                          handlers.HandleRoundStarted,
		sharedevents.RoundTagLookupFound:                  handlers.HandleTagNumberFound,
		sharedevents.RoundTagLookupNotFound:               handlers.HandleTagNumberNotFound,
		roundevents.RoundEntityCreated:                    handlers.HandleRoundEntityCreated,
		roundevents.RoundParticipantStatusUpdateRequest:   handlers.HandleParticipantStatusUpdateRequest,
		roundevents.RoundParticipantDeclined:              handlers.HandleParticipantDeclined,
		roundevents.RoundEventMessageIDUpdate:             handlers.HandleRoundEventMessageIDUpdate,
		roundevents.RoundEventMessageIDUpdated:            handlers.HandleDiscordMessageIDUpdated,
		roundevents.RoundParticipantScoreUpdated:          handlers.HandleParticipantScoreUpdated,
		sharedevents.TagUpdateForScheduledRounds:          handlers.HandleScheduledRoundTagUpdate,
		roundevents.GetRoundRequest:                       handlers.HandleGetRoundRequest,
		roundevents.RoundUpdated:                          handlers.HandleRoundScheduleUpdate,
	}

	for topic, handlerFunc := range eventsToHandlers {
		handlerName := fmt.Sprintf("round.%s", topic)
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
					publishTopic := m.Metadata.Get("topic")
					if publishTopic != "" {
						// Add specific logging for round reminder messages
						if publishTopic == roundevents.DiscordRoundReminder {
							r.logger.InfoContext(ctx, "🚀 Publishing Discord Round Reminder",
								attr.String("original_message_id", msg.UUID),
								attr.String("new_message_id", m.UUID),
								attr.String("topic", publishTopic),
								attr.String("handler_name", handlerName),
							)
						} else {
							r.logger.InfoContext(ctx, "🚀 Auto-publishing message from handler return",
								attr.String("message_id", m.UUID),
								attr.String("topic", publishTopic),
							)
						}

						if err := r.publisher.Publish(publishTopic, m); err != nil {
							r.logger.ErrorContext(ctx, "Failed to publish message from handler return", attr.String("message_id", m.UUID), attr.String("topic", publishTopic), attr.Error(err))
							return nil, fmt.Errorf("failed to publish to %s: %w", publishTopic, err)
						}
					} else {
						r.logger.Warn("⚠️ Message returned by handler missing topic metadata, dropping", attr.String("message_id", msg.UUID))
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
