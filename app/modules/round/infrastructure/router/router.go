package roundrouter

import (
	"context"
	"log/slog"
	"os"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"

	roundhandlers "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/handlers"

	"github.com/ThreeDotsLabs/watermill/components/metrics"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
)

const (
	TestEnvironmentFlag  = "APP_ENV"
	TestEnvironmentValue = "test"
)

type RoundRouter struct {
	logger     *slog.Logger
	Router     *message.Router
	subscriber eventbus.EventBus
	publisher  eventbus.EventBus
	helper     utils.Helpers
	tracer     trace.Tracer

	metricsBuilder *metrics.PrometheusMetricsBuilder
	metricsEnabled bool
}

func NewRoundRouter(
	logger *slog.Logger,
	router *message.Router,
	subscriber eventbus.EventBus,
	publisher eventbus.EventBus,
	helper utils.Helpers,
	tracer trace.Tracer,
	registry *prometheus.Registry,
) *RoundRouter {
	actualAppEnv := os.Getenv(TestEnvironmentFlag)
	inTestEnv := actualAppEnv == TestEnvironmentValue

	var metricsBuilder *metrics.PrometheusMetricsBuilder
	if registry != nil && !inTestEnv {
		b := metrics.NewPrometheusMetricsBuilder(registry, "", "")
		metricsBuilder = &b
	}

	return &RoundRouter{
		logger:         logger,
		Router:         router,
		subscriber:     subscriber,
		publisher:      publisher,
		helper:         helper,
		tracer:         tracer,
		metricsBuilder: metricsBuilder,
		metricsEnabled: metricsBuilder != nil,
	}
}

func (r *RoundRouter) Configure(_ context.Context, handlers roundhandlers.Handlers) error {
	if r.metricsEnabled && r.metricsBuilder != nil {
		r.metricsBuilder.AddPrometheusRouterMetrics(r.Router)
	}

	// Register all handlers
	r.registerHandlers(handlers)
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
	handlerName := "round." + topic

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

func (r *RoundRouter) registerHandlers(h roundhandlers.Handlers) error {
	var metrics handlerwrapper.ReturningMetrics // reserved for Phase 6

	deps := handlerDeps{
		router:     r.Router,
		subscriber: r.subscriber,
		publisher:  r.publisher,
		logger:     r.logger,
		tracer:     r.tracer,
		helper:     r.helper,
		metrics:    metrics,
	}

	registerHandler(deps, roundevents.ScorecardUploadedV1, h.HandleScorecardUploaded)
	registerHandler(deps, roundevents.ScorecardURLRequestedV1, h.HandleScorecardURLRequested)
	registerHandler(deps, roundevents.ScorecardParseRequestedV1, h.HandleParseScorecardRequest)
	registerHandler(deps, roundevents.ScorecardParsedForNormalizationV1, h.HandleScorecardParsedForNormalization)
	registerHandler(deps, roundevents.ScorecardNormalizedV1, h.HandleScorecardNormalized)
	registerHandler(deps, roundevents.ImportCompletedV1, h.HandleImportCompleted)

	registerHandler(deps, roundevents.RoundCreationRequestedV1, h.HandleCreateRoundRequest)
	registerHandler(deps, roundevents.RoundEntityCreatedV1, h.HandleRoundEntityCreated)
	registerHandler(deps, roundevents.RoundEventMessageIDUpdateV1, h.HandleRoundEventMessageIDUpdate)

	registerHandler(deps, roundevents.RoundDeleteRequestedV1, h.HandleRoundDeleteRequest)
	registerHandler(deps, roundevents.RoundDeleteValidatedV1, h.HandleRoundDeleteValidated)
	registerHandler(deps, roundevents.RoundDeleteAuthorizedV1, h.HandleRoundDeleteAuthorized)

	registerHandler(deps, roundevents.RoundUpdateRequestedV1, h.HandleRoundUpdateRequest)
	registerHandler(deps, roundevents.RoundUpdateValidatedV1, h.HandleRoundUpdateValidated)
	registerHandler(deps, roundevents.RoundScheduleUpdatedV1, h.HandleRoundScheduleUpdate)

	registerHandler(deps, roundevents.RoundScoreUpdateRequestedV1, h.HandleScoreUpdateRequest)
	registerHandler(deps, roundevents.RoundScoreBulkUpdateRequestedV1, h.HandleScoreBulkUpdateRequest)
	registerHandler(deps, roundevents.RoundScoreUpdateValidatedV1, h.HandleScoreUpdateValidated)
	registerHandler(deps, roundevents.RoundParticipantScoreUpdatedV1, h.HandleParticipantScoreUpdated)

	registerHandler(deps, roundevents.RoundAllScoresSubmittedV1, h.HandleAllScoresSubmitted)
	registerHandler(deps, roundevents.RoundFinalizedV1, h.HandleRoundFinalized)
	// Register the minimal 'start requested' command: worker/scheduler will publish
	// RoundStartRequestedV1. The handler will consult the DB and perform domain logic.
	registerHandler(deps, roundevents.RoundStartRequestedV1, h.HandleRoundStartRequested)

	registerHandler(deps, roundevents.RoundParticipantJoinRequestedV1, h.HandleParticipantJoinRequest)
	registerHandler(deps, roundevents.RoundParticipantJoinValidationRequestedV1, h.HandleParticipantJoinValidationRequest)
	registerHandler(deps, roundevents.RoundParticipantStatusUpdateRequestedV1, h.HandleParticipantStatusUpdateRequest)
	registerHandler(deps, roundevents.RoundParticipantRemovalRequestedV1, h.HandleParticipantRemovalRequest)
	registerHandler(deps, roundevents.RoundParticipantDeclinedV1, h.HandleParticipantDeclined)

	registerHandler(deps, sharedevents.RoundTagLookupFoundV1, h.HandleTagNumberFound)
	// Subscribe to both the legacy leaderboard-prefixed not-found topic and the
	// canonical shared not-found topic so we handle replies regardless of which
	// subject the leaderboard service publishes under during the migration.
	registerHandler(deps, sharedevents.RoundTagNumberNotFoundV1, h.HandleTagNumberNotFound)
	registerHandler(deps, sharedevents.RoundTagLookupNotFoundV1, h.HandleTagNumberNotFound)
	registerHandler(deps, sharedevents.GetTagNumberFailedV1, h.HandleTagNumberLookupFailed)
	// Listen for leaderboard tag update events for scheduled rounds. The leaderboard
	// service publishes TagUpdateForScheduledRoundsV1 when player tag numbers change;
	// the round service should consume that topic to update upcoming rounds.
	registerHandler(deps, sharedevents.SyncRoundsTagRequestV1, h.HandleScheduledRoundTagSync)

	registerHandler(deps, roundevents.GetRoundRequestedV1, h.HandleGetRoundRequest)
	registerHandler(deps, roundevents.RoundReminderScheduledV1, h.HandleRoundReminder)
	registerHandler(deps, roundevents.RoundEventMessageIDUpdatedV1, h.HandleDiscordMessageIDUpdated)

	// PWA request/reply handlers (with wildcard for guild_id)
	registerHandler(deps, "round.list.request.v1.>", h.HandleRoundListRequest)

	return nil
}

func (r *RoundRouter) Close() error {
	return r.Router.Close()
}
