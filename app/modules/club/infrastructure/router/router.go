package clubrouter

import (
	"context"
	"log/slog"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	clubevents "github.com/Black-And-White-Club/frolf-bot-shared/events/club"
	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	clubmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/club"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	clubhandlers "github.com/Black-And-White-Club/frolf-bot/app/modules/club/infrastructure/handlers"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel/trace"
)

// ClubRouter handles Watermill handler registration for club events.
type ClubRouter struct {
	logger     *slog.Logger
	router     *message.Router
	subscriber eventbus.EventBus
	publisher  eventbus.EventBus
	helper     utils.Helpers
	tracer     trace.Tracer
	metrics    clubmetrics.ClubMetrics
}

// NewClubRouter creates a new ClubRouter.
func NewClubRouter(
	logger *slog.Logger,
	router *message.Router,
	subscriber eventbus.EventBus,
	publisher eventbus.EventBus,
	helper utils.Helpers,
	tracer trace.Tracer,
	metrics clubmetrics.ClubMetrics,
) *ClubRouter {
	return &ClubRouter{
		logger:     logger,
		router:     router,
		subscriber: subscriber,
		publisher:  publisher,
		helper:     helper,
		tracer:     tracer,
		metrics:    metrics,
	}
}

// Configure sets up the router with handlers.
func (r *ClubRouter) Configure(_ context.Context, handlers clubhandlers.Handlers) error {
	r.registerHandlers(handlers)
	return nil
}

// handlerDeps bundles dependencies for handler registration.
type handlerDeps struct {
	router     *message.Router
	subscriber eventbus.EventBus
	publisher  eventbus.EventBus
	logger     *slog.Logger
	tracer     trace.Tracer
	helper     utils.Helpers
	metrics    clubmetrics.ClubMetrics
}

// registerHandlers wires NATS topics to handler methods.
func (r *ClubRouter) registerHandlers(handlers clubhandlers.Handlers) {
	deps := handlerDeps{
		router:     r.router,
		subscriber: r.subscriber,
		publisher:  r.publisher,
		logger:     r.logger,
		tracer:     r.tracer,
		helper:     r.helper,
		metrics:    r.metrics,
	}

	r.logger.Info("Registering club module handlers",
		slog.String("club_info_request_subject", clubevents.ClubInfoRequestV2+".*"),
		slog.String("challenge_list_request_subject", clubevents.ChallengeListRequestV1+".*"),
		slog.String("challenge_detail_request_subject", clubevents.ChallengeDetailRequestV1+".*"),
		slog.String("guild_setup_subject", guildevents.GuildSetupRequestedV1),
		slog.String("club_sync_subject", sharedevents.ClubSyncFromDiscordRequestedV2),
	)

	registerHandler(deps, clubevents.ClubInfoRequestV2+".*", handlers.HandleClubInfoRequest)
	registerHandler(deps, clubevents.ChallengeListRequestV1+".*", handlers.HandleChallengeListRequest)
	registerHandler(deps, clubevents.ChallengeDetailRequestV1+".*", handlers.HandleChallengeDetailRequest)
	registerHandler(deps, clubevents.ChallengeOpenRequestedV1, handlers.HandleChallengeOpenRequested)
	registerHandler(deps, clubevents.ChallengeRespondRequestedV1, handlers.HandleChallengeRespondRequested)
	registerHandler(deps, clubevents.ChallengeWithdrawRequestedV1, handlers.HandleChallengeWithdrawRequested)
	registerHandler(deps, clubevents.ChallengeHideRequestedV1, handlers.HandleChallengeHideRequested)
	registerHandler(deps, clubevents.ChallengeRoundLinkRequestedV1, handlers.HandleChallengeRoundLinkRequested)
	registerHandler(deps, clubevents.ChallengeRoundUnlinkRequestedV1, handlers.HandleChallengeRoundUnlinkRequested)
	registerHandler(deps, clubevents.ChallengeMessageBindRequestedV1, handlers.HandleChallengeMessageBindRequested)
	registerHandler(deps, clubevents.ChallengeExpireRequestedV1, handlers.HandleChallengeExpireRequested)
	registerHandler(deps, roundevents.RoundFinalizedV2, handlers.HandleRoundFinalized)
	registerHandler(deps, roundevents.RoundDeletedV2, handlers.HandleRoundDeleted)
	registerHandler(deps, leaderboardevents.LeaderboardTagUpdatedV2, handlers.HandleLeaderboardTagUpdated)
	registerHandler(deps, guildevents.GuildSetupRequestedV1, handlers.HandleGuildSetup)
	registerHandler(deps, sharedevents.ClubSyncFromDiscordRequestedV2, handlers.HandleClubSyncFromDiscord)

	r.logger.Info("Club module handlers registered successfully")
}

// registerHandler is a generic function for type-safe Watermill handler registration.
func registerHandler[T any](
	deps handlerDeps,
	topic string,
	handler func(context.Context, *T) ([]handlerwrapper.Result, error),
) {
	handlerName := "club." + topic

	deps.router.AddHandler(
		handlerName,
		topic,
		deps.subscriber,
		"",
		deps.publisher,
		handlerwrapper.WrapTransformingTyped(
			handlerName,
			deps.logger,
			deps.tracer,
			deps.helper,
			newReturningMetricsAdapter(deps.metrics),
			handler,
		),
	)
}

type returningMetricsAdapter struct {
	metrics clubmetrics.ClubMetrics
}

func newReturningMetricsAdapter(metrics clubmetrics.ClubMetrics) handlerwrapper.ReturningMetrics {
	if metrics == nil {
		return nil
	}
	return &returningMetricsAdapter{metrics: metrics}
}

func (a *returningMetricsAdapter) RecordAttempt(ctx context.Context, handler string) {
	a.metrics.RecordHandlerAttempt(ctx, handler)
}

func (a *returningMetricsAdapter) RecordSuccess(ctx context.Context, handler string) {
	a.metrics.RecordHandlerSuccess(ctx, handler)
}

func (a *returningMetricsAdapter) RecordFailure(ctx context.Context, handler string) {
	a.metrics.RecordHandlerFailure(ctx, handler)
}

func (a *returningMetricsAdapter) RecordDuration(ctx context.Context, handler string, d time.Duration) {
	a.metrics.RecordHandlerDuration(ctx, handler, d)
}

// Close shuts down the router.
func (r *ClubRouter) Close() error {
	return r.router.Close()
}
