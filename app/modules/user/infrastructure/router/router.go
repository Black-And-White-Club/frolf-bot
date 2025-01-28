package userrouter

import (
	"context"
	"fmt"
	"log/slog"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
	userhandlers "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/handlers"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// UserRouter handles routing for user module events.
type UserRouter struct {
	logger     *slog.Logger
	router     *message.Router
	subscriber message.Subscriber
}

// NewUserRouter creates a new UserRouter.
func NewUserRouter(logger *slog.Logger, router *message.Router, subscriber message.Subscriber) *UserRouter {
	return &UserRouter{
		logger:     logger,
		router:     router,
		subscriber: subscriber,
	}
}

// Configure sets up the router with the necessary handlers and dependencies.
func (r *UserRouter) Configure(userService userservice.Service) error {
	userHandlers := userhandlers.NewUserHandlers(userService, r.logger)

	// Add middleware to enhance logging and deduplication
	r.router.AddMiddleware(
		middleware.Recoverer,
		middleware.CorrelationID,
		r.debugMiddleware(),
	)

	if err := r.RegisterHandlers(context.Background(), userHandlers); err != nil {
		return fmt.Errorf("failed to register handlers: %w", err)
	}
	return nil
}

// RegisterHandlers sets up the mapping of events to handler functions.
func (r *UserRouter) RegisterHandlers(ctx context.Context, handlers userhandlers.Handlers) error {
	r.logger.Info("Entering RegisterHandlers")

	// Define the mapping of event topics to handler functions
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

	// Register each handler in the router
	for topic, handlerFunc := range eventsToHandlers {
		handlerName := fmt.Sprintf("user.module.handle.%s.%s", topic, watermill.NewUUID())
		r.logger.Info("Registering handler with AddNoPublisherHandler",
			slog.String("topic", topic),
			slog.String("handler", handlerName),
		)

		// Add the handler
		r.router.AddNoPublisherHandler(
			handlerName,  // Handler name
			topic,        // Topic
			r.subscriber, // Subscriber (EventBus)
			handlerFunc,  // Handler function
		)

		r.logger.Info("Handler registered successfully",
			slog.String("handler", handlerName),
			slog.String("topic", topic),
		)
	}

	r.logger.Info("Exiting RegisterHandlers")
	return nil
}

// debugMiddleware adds logging for all incoming messages.
func (r *UserRouter) debugMiddleware() message.HandlerMiddleware {
	return func(h message.HandlerFunc) message.HandlerFunc {
		return func(msg *message.Message) ([]*message.Message, error) {
			r.logger.Debug("Processing message in middleware",
				slog.String("topic", msg.Metadata.Get("topic")),
				slog.String("message_id", msg.UUID),
				slog.Any("metadata", msg.Metadata),
			)
			return h(msg)
		}
	}
}

// Close stops the router and cleans up resources.
func (r *UserRouter) Close() error {
	r.logger.Info("Closing UserRouter")
	return nil
}
