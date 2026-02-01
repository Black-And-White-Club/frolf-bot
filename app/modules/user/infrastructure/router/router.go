package userrouter

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
	userhandlers "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/handlers"
	"github.com/Black-And-White-Club/frolf-bot/config"
	"github.com/ThreeDotsLabs/watermill/components/metrics"
	"github.com/ThreeDotsLabs/watermill/message"
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

	// Register the event handlers with the router, passing the routerCtx.
	if err := r.registerHandlers(routerCtx, userHandlers); err != nil {
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
	handlerName := "user." + topic

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

// registerHandlers registers all user module handlers using the generic pattern
func (r *UserRouter) registerHandlers(ctx context.Context, handlers userhandlers.Handlers) error {
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

	// Register all user module handlers
	registerHandler(deps, userevents.UserSignupRequestedV1, handlers.HandleUserSignupRequest)
	registerHandler(deps, userevents.UserRoleUpdateRequestedV1, handlers.HandleUserRoleUpdateRequest)
	registerHandler(deps, userevents.GetUserRequestedV1, handlers.HandleGetUserRequest)
	registerHandler(deps, userevents.GetUserRoleRequestedV1, handlers.HandleGetUserRoleRequest)
	registerHandler(deps, sharedevents.TagUnavailableV1, handlers.HandleTagUnavailable)
	registerHandler(deps, sharedevents.TagAvailableV1, handlers.HandleTagAvailable)
	registerHandler(deps, userevents.UpdateUDiscIdentityRequestedV1, handlers.HandleUpdateUDiscIdentityRequest)
	registerHandler(deps, roundevents.ScorecardParsedV1, handlers.HandleScorecardParsed)
	registerHandler(deps, userevents.UserProfileUpdatedV1, handlers.HandleUserProfileUpdated)

	return nil
}

// Close stops the router.
func (r *UserRouter) Close() error {
	// Closing the Watermill router will also handle closing any decorated publishers/subscribers
	// that were added via AddHandler.
	return r.Router.Close()
}
