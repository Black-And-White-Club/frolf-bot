// Modified userrouter package to handle metrics properly in tests
package userrouter

import (
	"context"
	"fmt"
	"log/slog"
	"os" // Import os for environment variable check

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	tracingfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/tracing"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
	userhandlers "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/handlers"
	"github.com/Black-And-White-Club/frolf-bot/config"
	"github.com/ThreeDotsLabs/watermill/components/metrics" // Import watermill metrics
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/prometheus/client_golang/prometheus" // Import prometheus
	"go.opentelemetry.io/otel/trace"
)

const (
	// TestEnvironmentFlag is the flag to check if we're in a test environment
	TestEnvironmentFlag  = "APP_ENV"
	TestEnvironmentValue = "test"
)

// UserRouter handles routing for user module events.
type UserRouter struct {
	logger             *slog.Logger
	Router             *message.Router
	subscriber         eventbus.EventBus
	publisher          eventbus.EventBus
	config             *config.Config
	helper             utils.Helpers
	tracer             trace.Tracer
	middlewareHelper   utils.MiddlewareHelpers
	metricsBuilder     *metrics.PrometheusMetricsBuilder // Store a pointer to metrics builder
	prometheusRegistry *prometheus.Registry              // Store the registry pointer
	metricsEnabled     bool                              // Flag to track if metrics are enabled
}

// NewUserRouter creates a new UserRouter.
// It accepts a Prometheus Registry and creates its own metrics builder.
func NewUserRouter(
	logger *slog.Logger,
	router *message.Router,
	subscriber eventbus.EventBus,
	publisher eventbus.EventBus,
	config *config.Config,
	helper utils.Helpers,
	tracer trace.Tracer,
	prometheusRegistry *prometheus.Registry, // Accept Prometheus Registry pointer
) *UserRouter {
	// Check if we're in test environment - don't use metrics in tests
	inTestEnv := os.Getenv(TestEnvironmentFlag) == TestEnvironmentValue

	var metricsBuilder *metrics.PrometheusMetricsBuilder
	if prometheusRegistry != nil && !inTestEnv {
		// Only create the metrics builder if we have a valid registry and not in test env
		builder := metrics.NewPrometheusMetricsBuilder(prometheusRegistry, "", "")
		metricsBuilder = &builder
	}

	return &UserRouter{
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
		metricsEnabled:     prometheusRegistry != nil && !inTestEnv,
	}
}

// Configure sets up the router.
// It conditionally adds metrics middleware based on the environment.
func (r *UserRouter) Configure(userService userservice.Service, eventbus eventbus.EventBus, userMetrics usermetrics.UserMetrics) error {
	// Only add metrics if they're enabled
	if r.metricsEnabled && r.metricsBuilder != nil {
		r.logger.Info("Adding Prometheus router metrics middleware")
		r.metricsBuilder.AddPrometheusRouterMetrics(r.Router)
	} else {
		r.logger.Info("Skipping Prometheus router metrics middleware - either in test environment or metrics not configured")
	}

	// Create user handlers with logger and tracer
	userHandlers := userhandlers.NewUserHandlers(userService, r.logger, r.tracer, r.helper, userMetrics)

	r.Router.AddMiddleware(
		middleware.CorrelationID,
		r.middlewareHelper.CommonMetadataMiddleware("user"),
		r.middlewareHelper.DiscordMetadataMiddleware(),
		r.middlewareHelper.RoutingMetadataMiddleware(),
		middleware.Recoverer,
		middleware.Retry{MaxRetries: 3}.Middleware,
		tracingfrolfbot.TraceHandler(r.tracer),
	)

	// Register the event handlers with the router.
	if err := r.RegisterHandlers(context.Background(), userHandlers); err != nil {
		return fmt.Errorf("failed to register handlers: %w", err)
	}
	return nil
}

// RegisterHandlers registers event handlers.
func (r *UserRouter) RegisterHandlers(ctx context.Context, handlers userhandlers.Handlers) error {
	r.logger.InfoContext(ctx, "Entering Register Handlers for User")

	// Map event topics to their corresponding handler functions.
	eventsToHandlers := map[string]message.HandlerFunc{
		userevents.UserSignupRequest:           handlers.HandleUserSignupRequest,
		userevents.TagAvailable:                handlers.HandleTagAvailable,
		userevents.TagUnavailable:              handlers.HandleTagUnavailable,
		userevents.UserRoleUpdateRequest:       handlers.HandleUserRoleUpdateRequest,
		userevents.GetUserRoleRequest:          handlers.HandleGetUserRoleRequest,
		userevents.GetUserRequest:              handlers.HandleGetUserRequest,
		userevents.UserPermissionsCheckRequest: handlers.HandleGetUserRoleRequest,
	}
	r.logger.InfoContext(ctx, "Registering handlers for user module",
		attr.String("TagAvailable_constant", userevents.TagAvailable))

	for topic, handlerFunc := range eventsToHandlers {
		r.logger.InfoContext(ctx, "Setting up subscription",
			attr.String("topic", topic),
			attr.String("handler_name", fmt.Sprintf("user.%s", topic)))
		handlerName := fmt.Sprintf("user.%s", topic) // Unique name for the handler in the router

		r.Router.AddHandler(
			handlerName,  // Unique name for the handler
			topic,        // Topic to subscribe to
			r.subscriber, // Subscriber to use for this topic
			"",           // Output topic (empty string for no direct output)
			nil,          // Manual publisher (nil means use the default router publisher)
			func(msg *message.Message) ([]*message.Message, error) {
				messages, err := handlerFunc(msg)
				if err != nil {
					// Log and return the error if the handler logic failed.
					r.logger.ErrorContext(ctx, "Error processing message", attr.String("message_id", msg.UUID), attr.Any("error", err))
					return nil, err
				}

				// Iterate over any messages returned by the handler and publish them.
				for _, m := range messages {
					publishTopic := m.Metadata.Get("topic") // Get the intended output topic from metadata
					if publishTopic != "" {
						r.logger.InfoContext(ctx, "🚀 Auto-publishing message", attr.String("message_id", m.UUID), attr.String("topic", publishTopic))
						// Use the publisher provided to the router to publish the message.
						if err := r.publisher.Publish(publishTopic, m); err != nil {
							// If publishing fails, return an error to the router.
							return nil, fmt.Errorf("failed to publish to %s: %w", publishTopic, err)
						}
					} else {
						// Log a warning if a message is returned without a topic for auto-publishing.
						r.logger.Warn("⚠️ Message missing topic metadata, dropping", attr.String("message_id", msg.UUID))
					}
				}
				// Return nil messages and nil error to signal successful handling and publishing (if any).
				return nil, nil
			},
		)
	}
	return nil
}

// Close stops the router.
func (r *UserRouter) Close() error {
	// Closing the Watermill router will also handle closing any decorated publishers/subscribers
	// that were added via AddHandler.
	return r.Router.Close()
}
