package userrouter

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
	userhandlers "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/handlers"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// UserRouter handles routing for user module events.
type UserRouter struct {
	logger           *slog.Logger
	Router           *message.Router
	subscriber       eventbus.EventBus
	helper           utils.Helpers
	middlewareHelper utils.MiddlewareHelpers
}

// NewUserRouter creates a new UserRouter.
func NewUserRouter(logger *slog.Logger, router *message.Router, subscriber eventbus.EventBus, helper utils.Helpers) *UserRouter {
	return &UserRouter{
		logger:           logger,
		Router:           router,
		subscriber:       subscriber,
		helper:           helper,
		middlewareHelper: utils.NewMiddlewareHelper(),
	}
}

// Configure sets up the router.
func (r *UserRouter) Configure(userService userservice.Service,
) error {
	userHandlers := userhandlers.NewUserHandlers(userService, r.logger)

	r.Router.AddMiddleware(
		middleware.CorrelationID,
		r.middlewareHelper.CommonMetadataMiddleware("user"),
		r.middlewareHelper.DiscordMetadataMiddleware(),
		r.middlewareHelper.RoutingMetadataMiddleware(),
		middleware.Recoverer,
		middleware.Retry{MaxRetries: 3}.Middleware,
	)

	if err := r.RegisterHandlers(context.Background(), userHandlers); err != nil {
		return fmt.Errorf("failed to register handlers: %w", err)
	}

	return nil
}

// RegisterHandlers registers event handlers.
func (r *UserRouter) RegisterHandlers(ctx context.Context, handlers userhandlers.Handlers) error {
	eventsToHandlers := map[string]message.NoPublishHandlerFunc{
		userevents.UserSignupRequest:            handlers.HandleUserSignupRequest,
		userevents.TagAvailable:                 handlers.HandleTagAvailable,
		userevents.TagUnavailable:               handlers.HandleTagUnavailable,
		userevents.UserRoleUpdateRequest:        handlers.HandleUserRoleUpdateRequest,
		userevents.GetUserRoleRequest:           handlers.HandleGetUserRoleRequest,
		userevents.GetUserRequest:               handlers.HandleGetUserRequest,
		userevents.UserPermissionsCheckRequest:  handlers.HandleUserPermissionsCheckRequest,
		userevents.UserPermissionsCheckResponse: handlers.HandleUserPermissionsCheckResponse,
		userevents.UserPermissionsCheckFailed:   handlers.HandleUserPermissionsCheckFailed,
	}

	for topic, handlerFunc := range eventsToHandlers {
		handlerName := fmt.Sprintf("user.%s", topic)
		r.Router.AddNoPublisherHandler(
			handlerName,
			topic,
			r.subscriber,
			handlerFunc,
		)
	}
	return nil
}
