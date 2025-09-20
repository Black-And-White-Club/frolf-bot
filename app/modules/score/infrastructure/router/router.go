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

// RegisterHandlers registers event handlers using the provided context.
func (r *ScoreRouter) RegisterHandlers(ctx context.Context, handlers scorehandlers.Handlers) error {
	eventsToHandlers := map[string]message.HandlerFunc{
		scoreevents.ProcessRoundScoresRequest: handlers.HandleProcessRoundScoresRequest,
		scoreevents.ScoreUpdateRequest:        handlers.HandleCorrectScoreRequest,
		scoreevents.ScoreBulkUpdateRequest:    handlers.HandleBulkCorrectScoreRequest,
		// Reprocessing triggers after overrides
		scoreevents.ScoreUpdateSuccess:     handlers.HandleReprocessAfterScoreUpdate,
		scoreevents.ScoreBulkUpdateSuccess: handlers.HandleReprocessAfterScoreUpdate,
	}

	for topic, handlerFunc := range eventsToHandlers {
		handlerName := fmt.Sprintf("score.%s", topic)
		r.Router.AddHandler(
			handlerName,
			topic,
			r.subscriber,
			"",
			nil,
			func(msg *message.Message) ([]*message.Message, error) {
				messages, err := handlerFunc(msg)
				if err != nil {
					r.logger.ErrorContext(ctx, "Error processing message", attr.String("message_id", msg.UUID), attr.Any("error", err))
					return nil, err
				}
				for _, m := range messages {
					publishTopic := m.Metadata.Get("topic")
					if publishTopic != "" {
						// Pass the context from RegisterHandlers (routerCtx) to Publish
						if err := r.publisher.Publish(publishTopic, m); err != nil {
							return nil, fmt.Errorf("failed to publish to %s: %w", publishTopic, err)
						}
					}
				}
				return nil, nil
			},
		)
	}
	return nil
}

func (r *ScoreRouter) Close() error {
	return r.Router.Close()
}
