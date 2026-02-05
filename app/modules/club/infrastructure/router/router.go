package clubrouter

import (
	"context"
	"log/slog"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	clubevents "github.com/Black-And-White-Club/frolf-bot-shared/events/club"
	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
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
}

// NewClubRouter creates a new ClubRouter.
func NewClubRouter(
	logger *slog.Logger,
	router *message.Router,
	subscriber eventbus.EventBus,
	publisher eventbus.EventBus,
	helper utils.Helpers,
	tracer trace.Tracer,
) *ClubRouter {
	return &ClubRouter{
		logger:     logger,
		router:     router,
		subscriber: subscriber,
		publisher:  publisher,
		helper:     helper,
		tracer:     tracer,
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
	}

	r.logger.Info("Registering club module handlers",
		slog.String("club_info_request_subject", clubevents.ClubInfoRequestV1+".*"),
		slog.String("guild_setup_subject", guildevents.GuildSetupRequestedV1),
		slog.String("user_signup_subject", userevents.UserSignupRequestedV1),
	)

	registerHandler(deps, clubevents.ClubInfoRequestV1+".*", handlers.HandleClubInfoRequest)
	registerHandler(deps, guildevents.GuildSetupRequestedV1, handlers.HandleGuildSetup)
	registerHandler(deps, userevents.UserSignupRequestedV1, handlers.HandleUserSignupRequest)

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
			nil,
			handler,
		),
	)
}

// Close shuts down the router.
func (r *ClubRouter) Close() error {
	return r.router.Close()
}
