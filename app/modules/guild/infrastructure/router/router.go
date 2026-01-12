package guildrouter

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	guildmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/guild"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	guildservice "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/application"
	guildhandlers "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/infrastructure/handlers"
	"github.com/Black-And-White-Club/frolf-bot/config"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.opentelemetry.io/otel/trace"
)

// GuildRouter handles routing for guild module events.
type GuildRouter struct {
	logger     *slog.Logger
	Router     *message.Router
	subscriber eventbus.EventBus
	publisher  eventbus.EventBus
	config     *config.Config
	helper     utils.Helpers
	tracer     trace.Tracer
}

// NewGuildRouter creates a new GuildRouter.
func NewGuildRouter(
	logger *slog.Logger,
	router *message.Router,
	subscriber eventbus.EventBus,
	publisher eventbus.EventBus,
	config *config.Config,
	helper utils.Helpers,
	tracer trace.Tracer,
) *GuildRouter {
	return &GuildRouter{
		logger:     logger,
		Router:     router,
		subscriber: subscriber,
		publisher:  publisher,
		config:     config,
		helper:     helper,
		tracer:     tracer,
	}
}

// Configure sets up the router with the necessary handlers and dependencies.
func (r *GuildRouter) Configure(routerCtx context.Context, guildService guildservice.Service, eventbus eventbus.EventBus, guildMetrics guildmetrics.GuildMetrics) error {
	guildHandlers := guildhandlers.NewGuildHandlers(guildService, r.logger, r.tracer, r.helper, guildMetrics)

	r.Router.AddMiddleware(
		middleware.CorrelationID,
		utils.NewMiddlewareHelper().CommonMetadataMiddleware("guild"),
		utils.NewMiddlewareHelper().DiscordMetadataMiddleware(),
		utils.NewMiddlewareHelper().RoutingMetadataMiddleware(),
		middleware.Recoverer,
	)

	if err := r.RegisterHandlers(routerCtx, guildHandlers); err != nil {
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

// registerHandler registers a pure transformation-pattern handler with typed payload.
func registerHandler[T any](
	deps handlerDeps,
	topic string,
	handler func(context.Context, *T) ([]handlerwrapper.Result, error),
) {
	handlerName := "guild." + topic

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

// RegisterHandlers registers event handlers using the pure transformation pattern.
func (r *GuildRouter) RegisterHandlers(ctx context.Context, handlers guildhandlers.Handlers) error {
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

	registerHandler(deps, guildevents.GuildConfigCreationRequestedV1, handlers.HandleCreateGuildConfig)
	registerHandler(deps, guildevents.GuildConfigRetrievalRequestedV1, handlers.HandleRetrieveGuildConfig)
	registerHandler(deps, guildevents.GuildConfigUpdateRequestedV1, handlers.HandleUpdateGuildConfig)
	registerHandler(deps, guildevents.GuildConfigDeletionRequestedV1, handlers.HandleDeleteGuildConfig)
	registerHandler(deps, guildevents.GuildSetupRequestedV1, handlers.HandleGuildSetup)

	return nil
}

// Close stops the router.
func (r *GuildRouter) Close() error {
	return r.Router.Close()
}
