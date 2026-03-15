package bettingrouter

import (
	"context"
	"log/slog"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	bettingevents "github.com/Black-And-White-Club/frolf-bot-shared/events/betting"
	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	bettinghandlers "github.com/Black-And-White-Club/frolf-bot/app/modules/betting/infrastructure/handlers"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel/trace"
)

type Router struct {
	logger     *slog.Logger
	router     *message.Router
	subscriber eventbus.EventBus
	publisher  eventbus.EventBus
	helper     utils.Helpers
	tracer     trace.Tracer
}

func NewRouter(
	logger *slog.Logger,
	router *message.Router,
	subscriber eventbus.EventBus,
	publisher eventbus.EventBus,
	helper utils.Helpers,
	tracer trace.Tracer,
) *Router {
	return &Router{
		logger:     logger,
		router:     router,
		subscriber: subscriber,
		publisher:  publisher,
		helper:     helper,
		tracer:     tracer,
	}
}

func (r *Router) Configure(_ context.Context, handlers bettinghandlers.Handlers) error {
	deps := handlerDeps{
		router:     r.router,
		subscriber: r.subscriber,
		publisher:  r.publisher,
		logger:     r.logger,
		tracer:     r.tracer,
		helper:     r.helper,
	}

	registerHandler(deps, roundevents.RoundFinalizedV2, handlers.HandleRoundFinalized)
	registerHandler(deps, roundevents.RoundDeletedV2, handlers.HandleRoundDeleted)
	// NATS request/reply: betting.snapshot.request.v1.> captures per-club subjects
	registerHandler(deps, bettingevents.BettingSnapshotRequestV1+".>", handlers.HandleBettingSnapshotRequest)
	// Suspend open markets when a club loses betting entitlement (freeze/disable).
	registerHandler(deps, guildevents.GuildFeatureAccessUpdatedV1, handlers.HandleFeatureAccessUpdated)
	// Mirror season-point awards into the betting wallet journal.
	registerHandler(deps, sharedevents.PointsAwardedV1, handlers.HandlePointsAwarded)

	return nil
}

type handlerDeps struct {
	router     *message.Router
	subscriber eventbus.EventBus
	publisher  eventbus.EventBus
	logger     *slog.Logger
	tracer     trace.Tracer
	helper     utils.Helpers
}

func registerHandler[T any](
	deps handlerDeps,
	topic string,
	handler func(context.Context, *T) ([]handlerwrapper.Result, error),
) {
	handlerName := "betting." + topic

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
			nil,
			handler,
		),
	)
}

func (r *Router) Close() error {
	return r.router.Close()
}
