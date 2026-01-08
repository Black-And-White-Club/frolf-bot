package scorerouter

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
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

// RegisterHandlers registers event handlers using V1 versioned event constants.
func (r *ScoreRouter) RegisterHandlers(ctx context.Context, handlers scorehandlers.Handlers) error {
	eventsToHandlers := map[string]message.HandlerFunc{
		// Score Processing Flow (from processing.go)
		scoreevents.ProcessRoundScoresRequestedV1: handlers.HandleProcessRoundScoresRequest,

		// Score Update Flow (from updates.go)
		// These handlers now trigger reprocessing directly, so we don't need separate subscriptions
		scoreevents.ScoreUpdateRequestedV1:     handlers.HandleCorrectScoreRequest,
		scoreevents.ScoreBulkUpdateRequestedV1: handlers.HandleBulkCorrectScoreRequest,
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
					// Router resolves topic (not metadata)
					publishTopic := r.getPublishTopic(handlerName, m)

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
				return nil, nil
			},
		)
	}
	return nil
}

func (r *ScoreRouter) Close() error {
	return r.Router.Close()
}

// getPublishTopic resolves the topic to publish for a given handler's returned message.
// Use explicit mapping where possible; fallback to metadata during migration for
// handlers that can emit multiple different output topics.
func (r *ScoreRouter) getPublishTopic(handlerName string, msg *message.Message) string {
	switch {
	case handlerName == "score."+scoreevents.ProcessRoundScoresRequestedV1:
		// HandleProcessRoundScoresRequest always publishes a leaderboard batch assignment request
		return sharedevents.LeaderboardBatchTagAssignmentRequestedV1

	case handlerName == "score."+scoreevents.ScoreUpdateRequestedV1:
		// CorrectScore may emit success or failure; use metadata fallback during migration
		return msg.Metadata.Get("topic")

	case handlerName == "score."+scoreevents.ScoreBulkUpdateRequestedV1:
		// Bulk updates can produce different topics; fallback to metadata
		return msg.Metadata.Get("topic")

	default:
		r.logger.Warn("unknown handler in topic resolution",
			attr.String("handler", handlerName),
		)
		return msg.Metadata.Get("topic")
	}
}
