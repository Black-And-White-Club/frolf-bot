package guildrouter

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	guildmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/guild"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
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

// RegisterHandlers registers event handlers using V1 versioned event constants.
func (r *GuildRouter) RegisterHandlers(ctx context.Context, handlers guildhandlers.Handlers) error {
	eventsToHandlers := map[string]message.HandlerFunc{
		// Guild Config Creation Flow (from config.go)
		guildevents.GuildConfigCreationRequestedV1: handlers.HandleCreateGuildConfig,

		// Guild Config Retrieval Flow (from config.go)
		guildevents.GuildConfigRetrievalRequestedV1: handlers.HandleRetrieveGuildConfig,

		// Guild Config Update Flow (from config.go)
		guildevents.GuildConfigUpdateRequestedV1: handlers.HandleUpdateGuildConfig,

		// Guild Config Deletion Flow (from config.go)
		guildevents.GuildConfigDeletionRequestedV1: handlers.HandleDeleteGuildConfig,

		// Guild Setup Flow (from config.go)
		guildevents.GuildSetupRequestedV1: handlers.HandleGuildSetup,
	}

	for topic, handlerFunc := range eventsToHandlers {
		handlerName := fmt.Sprintf("guild.%s", topic)
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
				return nil, nil
			},
		)
	}
	return nil
}

// Close stops the router.
func (r *GuildRouter) Close() error {
	return r.Router.Close()
}
