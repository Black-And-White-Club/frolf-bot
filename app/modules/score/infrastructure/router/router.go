package scorerouter

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	scoremetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/score"
	tracingfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/tracing"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	scoreservice "github.com/Black-And-White-Club/frolf-bot/app/modules/score/application"
	scorehandlers "github.com/Black-And-White-Club/frolf-bot/app/modules/score/infrastructure/handlers"
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

type ScoreRouter struct {
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

func NewScoreRouter(
	logger *slog.Logger,
	router *message.Router,
	subscriber eventbus.EventBus,
	publisher eventbus.EventBus,
	config *config.Config,
	helper utils.Helpers,
	tracer trace.Tracer,
	prometheusRegistry *prometheus.Registry,
) *ScoreRouter {
	inTestEnv := os.Getenv(TestEnvironmentFlag) == TestEnvironmentValue

	var metricsBuilder *metrics.PrometheusMetricsBuilder
	if prometheusRegistry != nil && !inTestEnv {
		builder := metrics.NewPrometheusMetricsBuilder(prometheusRegistry, "", "")
		metricsBuilder = &builder
	}
	return &ScoreRouter{
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

// Configure sets up the router using the provided context and score service.
// It registers handlers and adds middleware to the router held by the ScoreRouter.
func (r *ScoreRouter) Configure(routerCtx context.Context, scoreService scoreservice.Service, eventbus eventbus.EventBus, scoreMetrics scoremetrics.ScoreMetrics) error {
	if r.metricsEnabled && r.metricsBuilder != nil {
		r.logger.Info("Adding Prometheus router metrics middleware")
		r.metricsBuilder.AddPrometheusRouterMetrics(r.Router)
	} else {
		r.logger.Info("Skipping Prometheus router metrics middleware - either in test environment or metrics not configured")
	}

	scoreHandlers := scorehandlers.NewScoreHandlers(scoreService, r.logger, r.tracer, r.helper, scoreMetrics)

	r.Router.AddMiddleware(
		middleware.CorrelationID,
		r.middlewareHelper.CommonMetadataMiddleware("score"),
		r.middlewareHelper.DiscordMetadataMiddleware(),
		r.middlewareHelper.RoutingMetadataMiddleware(),
		middleware.Recoverer,
		middleware.Retry{MaxRetries: 3}.Middleware,
		tracingfrolfbot.TraceHandler(r.tracer),
	)

	// Pass the routerCtx to RegisterHandlers
	if err := r.RegisterHandlers(routerCtx, scoreHandlers); err != nil {
		return fmt.Errorf("failed to register handlers: %w", err)
	}
	return nil
}

// handlerDeps holds dependencies for handler registration
type handlerDeps struct {
	router     *message.Router
	logger     *slog.Logger
	tracer     trace.Tracer
	helper     utils.Helpers
	metrics    handlerwrapper.ReturningMetrics
	subscriber eventbus.EventBus
	publisher  eventbus.EventBus
}

// registerHandler registers a typed handler using the generic WrapTransformingTyped wrapper.
func registerHandler[T any](
	deps handlerDeps,
	topic string,
	handler func(context.Context, *T) ([]handlerwrapper.Result, error),
) {
	handlerName := fmt.Sprintf("score.%s", topic)

	deps.router.AddHandler(
		handlerName,
		topic,
		deps.subscriber,
		"", // No static output topic (each result has its own topic in metadata)
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

// RegisterHandlers registers event handlers using V1 versioned event constants.
func (r *ScoreRouter) RegisterHandlers(ctx context.Context, handlers scorehandlers.Handlers) error {
	// Note: metrics set to nil for now (reserved for future phase)
	// ScoreMetrics interface has the required handler methods, but we use the base interface
	var metrics handlerwrapper.ReturningMetrics

	deps := handlerDeps{
		router:     r.Router,
		logger:     r.logger,
		tracer:     r.tracer,
		helper:     r.helper,
		metrics:    metrics,
		subscriber: r.subscriber,
		publisher:  r.publisher,
	}

	// Register ProcessRoundScoresRequest handler
	registerHandler(
		deps,
		scoreevents.ProcessRoundScoresRequestedV1,
		handlers.HandleProcessRoundScoresRequest,
	)

	// Register CorrectScoreRequest handler
	registerHandler(
		deps,
		scoreevents.ScoreUpdateRequestedV1,
		handlers.HandleCorrectScoreRequest,
	)

	// Register BulkCorrectScoreRequest handler
	registerHandler(
		deps,
		scoreevents.ScoreBulkUpdateRequestedV1,
		handlers.HandleBulkCorrectScoreRequest,
	)

	r.logger.Info("Registered score handlers",
		attr.Int("handlers", 3),
	)

	return nil
}

func (r *ScoreRouter) Close() error {
	return r.Router.Close()
}
