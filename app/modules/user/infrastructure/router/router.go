// Modified userrouter package to handle metrics properly in tests
package userrouter

import (
	"context"
	"fmt"
	"log/slog"
	"os" // Import os for environment variable check

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	tracingfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/tracing"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
	userhandlers "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/handlers"
	"github.com/Black-And-White-Club/frolf-bot/config"
	"github.com/ThreeDotsLabs/watermill/components/metrics"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/prometheus/client_golang/prometheus"
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
	metricsBuilder     *metrics.PrometheusMetricsBuilder
	prometheusRegistry *prometheus.Registry
	metricsEnabled     bool // Flag to indicate if metrics are enabled
}

// NewUserRouter creates a new UserRouter.
// It initializes the metrics builder conditionally based on the environment and registry.
func NewUserRouter(
	logger *slog.Logger,
	router *message.Router,
	subscriber eventbus.EventBus,
	publisher eventbus.EventBus,
	config *config.Config,
	helper utils.Helpers,
	tracer trace.Tracer,
	prometheusRegistry *prometheus.Registry,
) *UserRouter {
	// Check if we're in test environment - don't use metrics in tests
	actualAppEnv := os.Getenv(TestEnvironmentFlag)
	logger.Info("NewUserRouter: Environment check",
		"APP_ENV_Actual", actualAppEnv,
		"TestEnvironmentValue", TestEnvironmentValue,
		"prometheusRegistryProvided", prometheusRegistry != nil,
	)

	inTestEnv := actualAppEnv == TestEnvironmentValue
	logger.Info("NewUserRouter: inTestEnv determined", "inTestEnv", inTestEnv)

	var metricsBuilder *metrics.PrometheusMetricsBuilder
	// Only create the metrics builder if a registry is provided AND we are NOT in the test environment
	if prometheusRegistry != nil && !inTestEnv {
		logger.Info("NewUserRouter: Creating Prometheus metrics builder")
		builder := metrics.NewPrometheusMetricsBuilder(prometheusRegistry, "", "")
		metricsBuilder = &builder
	} else {
		logger.Info("NewUserRouter: Skipping Prometheus metrics builder creation",
			"prometheusRegistryProvided", prometheusRegistry != nil,
			"inTestEnv", inTestEnv,
		)
	}

	// metricsEnabled is true only if a registry is provided AND we are NOT in the test environment
	metricsEnabled := prometheusRegistry != nil && !inTestEnv
	logger.Info("NewUserRouter: metricsEnabled determined", "metricsEnabled", metricsEnabled)

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
		metricsEnabled:     metricsEnabled, // Use the determined value
	}
}

// Configure sets up the router.
// It conditionally adds metrics middleware and registers handlers.
// It now accepts a context for the router run.
func (r *UserRouter) Configure(routerCtx context.Context, userService userservice.Service, eventbus eventbus.EventBus, userMetrics usermetrics.UserMetrics) error {
	// Add explicit debug logging that will definitely show up
	fmt.Printf("DEBUG: UserRouter.Configure() called - starting handler registration\n")

	// Add logging before the conditional check for adding middleware
	r.logger.Info("Configure: Checking metricsEnabled before adding middleware",
		"metricsEnabled", r.metricsEnabled,
		"metricsBuilderNil", r.metricsBuilder == nil,
	)

	// Only add metrics if they're enabled and the builder was created
	if r.metricsEnabled && r.metricsBuilder != nil {
		r.logger.Info("Adding Prometheus router metrics middleware for User")
		r.metricsBuilder.AddPrometheusRouterMetrics(r.Router)
	} else {
		// This log message confirms that metrics are being skipped in the test environment
		r.logger.Info("Skipping Prometheus router metrics middleware for User - either in test environment or metrics not configured")
	}

	// Create user handlers with logger, tracer, and metrics
	userHandlers := userhandlers.NewUserHandlers(userService, r.logger, r.tracer, r.helper, userMetrics)

	// Add common middleware
	r.Router.AddMiddleware(
		middleware.CorrelationID,
		r.middlewareHelper.CommonMetadataMiddleware("user"),
		r.middlewareHelper.DiscordMetadataMiddleware(),
		r.middlewareHelper.RoutingMetadataMiddleware(),
		middleware.Recoverer,
		middleware.Retry{MaxRetries: 3}.Middleware,
		tracingfrolfbot.TraceHandler(r.tracer),
	)

	// Register the event handlers with the router, passing the routerCtx.
	fmt.Printf("DEBUG: About to call RegisterHandlers() for user module\n")
	if err := r.RegisterHandlers(routerCtx, userHandlers); err != nil {
		return fmt.Errorf("failed to register handlers: %w", err)
	}
	fmt.Printf("DEBUG: RegisterHandlers() completed successfully for user module\n")
	return nil
}

// RegisterHandlers registers event handlers using V1 versioned event constants.
func (r *UserRouter) RegisterHandlers(ctx context.Context, handlers userhandlers.Handlers) error {
	fmt.Printf("DEBUG: RegisterHandlers() called for user module\n")
	r.logger.InfoContext(ctx, "Entering Register Handlers for User")

	// Map event topics to their corresponding handler functions.
	eventsToHandlers := map[string]message.HandlerFunc{
		// User Creation Flow (from creation.go)
		userevents.UserSignupRequestedV1: handlers.HandleUserSignupRequest,

		// Tag Availability Flow (from tags.go) - responses published to user stream by leaderboard
		userevents.TagAvailableV1:   handlers.HandleTagAvailable,
		userevents.TagUnavailableV1: handlers.HandleTagUnavailable,

		// Role Update Flow (from roles.go)
		userevents.UserRoleUpdateRequestedV1: handlers.HandleUserRoleUpdateRequest,

		// Role Retrieval Flow (from roles.go)
		userevents.GetUserRoleRequestedV1: handlers.HandleGetUserRoleRequest,

		// User Retrieval Flow (from retrieval.go)
		userevents.GetUserRequestedV1: handlers.HandleGetUserRequest,

		// Permissions Check Flow (from roles.go)
		userevents.UserPermissionsCheckRequestedV1: handlers.HandleGetUserRoleRequest,

		// UDisc Identity Flow (from udisc.go)
		userevents.UpdateUDiscIdentityRequestedV1: handlers.HandleUpdateUDiscIdentityRequest,

		// Cross-module: Scorecard parsed for user (from round/import.go)
		roundevents.ScorecardParsedForUserV1: handlers.HandleScorecardParsed,
	}
	r.logger.InfoContext(ctx, "Registering handlers for user module",
		attr.String("TagAvailable_constant", userevents.TagAvailableV1))

	for topic, handlerFunc := range eventsToHandlers {
		fmt.Printf("DEBUG: Registering handler for topic: %s\n", topic)
		// Use the context passed to RegisterHandlers
		r.logger.InfoContext(ctx, "Setting up subscription",
			attr.String("topic", topic),
			attr.String("handler_name", fmt.Sprintf("user.%s", topic)))
		handlerName := fmt.Sprintf("user.%s", topic)

		r.Router.AddHandler(
			handlerName,
			topic,        // Topic to subscribe to
			r.subscriber, // Subscriber to use for this topic
			"",           // Output topic (empty string for no direct output - manual publishing below)
			nil,          // Manual publisher (nil means use the default router publisher if configured, but we'll use r.publisher explicitly)
			func(msg *message.Message) ([]*message.Message, error) {
				messages, err := handlerFunc(msg)
				if err != nil {
					r.logger.ErrorContext(ctx, "Error processing message by handler", attr.String("message_id", msg.UUID), attr.Any("error", err))
					return nil, err
				}

				// Manually iterate over any messages returned by the handler and publish them
				for _, m := range messages {
					// Get the intended output topic from metadata
					publishTopic := m.Metadata.Get("topic")

					// INVARIANT: Topic must be resolvable
					if publishTopic == "" {
						r.logger.Error("router failed to resolve publish topic - MESSAGE DROPPED",
							attr.String("handler", handlerName),
							attr.String("msg_uuid", m.UUID),
							attr.String("correlation_id", m.Metadata.Get("correlation_id")),
						)
						// Skip publishing but don't fail entire batch
						continue
					}

					r.logger.InfoContext(ctx, "publishing message",
						attr.String("topic", publishTopic),
						attr.String("handler", handlerName),
						attr.String("correlation_id", m.Metadata.Get("correlation_id")),
					)

					if err := r.publisher.Publish(publishTopic, m); err != nil {
						return nil, fmt.Errorf("failed to publish to %s: %w", publishTopic, err)
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
