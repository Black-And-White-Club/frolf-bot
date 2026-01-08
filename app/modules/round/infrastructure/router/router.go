package roundrouter

import (
	"context"
	"fmt"
	"log/slog"
	"os" // Import os for environment variable check

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
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

// RegisterHandlers registers event handlers using V1 versioned event constants.
func (r *RoundRouter) RegisterHandlers(ctx context.Context, handlers roundhandlers.Handlers) error {
	r.logger.InfoContext(ctx, "Entering RegisterHandlers for Round")

	eventsToHandlers := map[string]message.HandlerFunc{
		// Round Creation Flow (from creation.go)
		roundevents.RoundCreationRequestedV1:     handlers.HandleCreateRoundRequest,
		roundevents.RoundEntityCreatedV1:         handlers.HandleRoundEntityCreated,
		roundevents.RoundEventMessageIDUpdateV1:  handlers.HandleRoundEventMessageIDUpdate,
		roundevents.RoundEventMessageIDUpdatedV1: handlers.HandleDiscordMessageIDUpdated,

		// Round Update Flow (from update.go)
		roundevents.RoundUpdateRequestedV1: handlers.HandleRoundUpdateRequest,
		roundevents.RoundUpdateValidatedV1: handlers.HandleRoundUpdateValidated,
		roundevents.RoundUpdatedV1:         handlers.HandleRoundScheduleUpdate,

		// Round Delete Flow (from delete.go)
		roundevents.RoundDeleteRequestedV1:  handlers.HandleRoundDeleteRequest,
		roundevents.RoundDeleteValidatedV1:  handlers.HandleRoundDeleteValidated,
		roundevents.RoundDeleteAuthorizedV1: handlers.HandleRoundDeleteAuthorized,

		// Round Participant Flow (from participants.go)
		roundevents.RoundParticipantJoinRequestedV1:           handlers.HandleParticipantJoinRequest,
		roundevents.RoundParticipantJoinValidationRequestedV1: handlers.HandleParticipantJoinValidationRequest,
		roundevents.RoundParticipantRemovalRequestedV1:        handlers.HandleParticipantRemovalRequest,
		roundevents.RoundParticipantStatusUpdateRequestedV1:   handlers.HandleParticipantStatusUpdateRequest,
		roundevents.RoundParticipantDeclinedV1:                handlers.HandleParticipantDeclined,

		// Round Scoring Flow (from scoring.go)
		roundevents.RoundScoreUpdateRequestedV1:    handlers.HandleScoreUpdateRequest,
		roundevents.RoundScoreUpdateValidatedV1:    handlers.HandleScoreUpdateValidated,
		roundevents.RoundParticipantScoreUpdatedV1: handlers.HandleParticipantScoreUpdated,
		roundevents.RoundAllScoresSubmittedV1:      handlers.HandleAllScoresSubmitted,

		// Round Lifecycle Flow (from lifecycle.go)
		roundevents.RoundFinalizedV1:         handlers.HandleRoundFinalized,
		roundevents.RoundReminderScheduledV1: handlers.HandleRoundReminder,
		roundevents.RoundStartedV1:           handlers.HandleRoundStarted,

		// Tag Lookup Flow - cross-module events (from shared/tags.go)
		sharedevents.RoundTagLookupFoundV1:         handlers.HandleTagNumberFound,
		sharedevents.RoundTagLookupNotFoundV1:      handlers.HandleTagNumberNotFound,
		leaderboardevents.GetTagNumberFailedV1:     handlers.HandleTagNumberLookupFailed,
		sharedevents.TagUpdateForScheduledRoundsV1: handlers.HandleScheduledRoundTagUpdate,

		// Round Retrieval Flow (from retrieval.go)
		roundevents.GetRoundRequestedV1: handlers.HandleGetRoundRequest,

		// Scorecard Import Flow (from import.go)
		roundevents.ScorecardUploadedV1:       handlers.HandleScorecardUploaded,
		roundevents.ScorecardURLRequestedV1:   handlers.HandleScorecardURLRequested,
		roundevents.ScorecardParseRequestedV1: handlers.HandleParseScorecardRequest,
		roundevents.ImportCompletedV1:         handlers.HandleImportCompleted,

		// Cross-module events (user module - from udisc.go)
		userevents.UDiscMatchConfirmedV1: handlers.HandleUserMatchConfirmedForIngest,
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

					// Add specific logging for round reminder messages
					if publishTopic == roundevents.RoundReminderScheduledV1 {
						r.logger.InfoContext(ctx, "Publishing Discord Round Reminder",
							attr.String("original_message_id", msg.UUID),
							attr.String("new_message_id", m.UUID),
							attr.String("topic", publishTopic),
							attr.String("handler_name", handlerName),
							attr.String("correlation_id", m.Metadata.Get("correlation_id")),
						)
					} else {
						r.logger.InfoContext(ctx, "publishing message",
							attr.String("message_id", m.UUID),
							attr.String("topic", publishTopic),
							attr.String("handler", handlerName),
							attr.String("correlation_id", m.Metadata.Get("correlation_id")),
						)
					}

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

// getPublishTopic resolves the topic to publish for a given handler's returned message.
// During the migration we keep the metadata fallback for handlers that can emit
// multiple different outcome events. As handlers are stabilized we can make
// explicit mappings here and remove the metadata fallback.
func (r *RoundRouter) getPublishTopic(handlerName string, msg *message.Message) string {
	// Router-owned topic resolution for round handlers.
	// For handlers that can emit multiple different outcome events we still
	// fallback to the message metadata during migration. For single-outcome
	// handlers we return the concrete topic here so routing is owned by the
	// router (not handler/helpers).
	switch handlerName {
	// Handler that updates a round with the Discord message ID and then
	// emits a scheduled/updated event for downstream consumers.
	case "round." + roundevents.RoundEventMessageIDUpdateV1:
		return roundevents.RoundEventMessageIDUpdatedV1

	// Round started handler publishes a Discord-specific start event
	case "round." + roundevents.RoundStartedV1:
		return roundevents.RoundStartedDiscordV1

	// When a round is finalized the router should forward to the score
	// processing request topic for downstream scoring consumers.
	case "round." + roundevents.RoundFinalizedV1:
		return roundevents.ProcessRoundScoresRequestedV1

	// Finalization and import handlers can emit multiple different topics
	// (discord vs backend vs errors). Preserve metadata fallback for those.
	case "round." + roundevents.RoundAllScoresSubmittedV1:
		return msg.Metadata.Get("topic")

	case "round." + roundevents.ProcessRoundScoresRequestedV1:
		return msg.Metadata.Get("topic")

	default:
		// Unknown or multi-outcome handlers: use metadata as a graceful fallback
		// while the migration completes.
		r.logger.Warn("unknown handler in round topic resolution, falling back to metadata",
			attr.String("handler", handlerName),
		)
		return msg.Metadata.Get("topic")
	}
}

// Close stops the router.
func (r *RoundRouter) Close() error {
	return r.Router.Close()
}
