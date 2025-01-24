package userrouter

import (
	"context"
	"fmt"
	"log/slog"

	userservice "github.com/Black-And-White-Club/tcr-bot/app/modules/user/application"
	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	userhandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/user/infrastructure/handlers"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
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
func (r *UserRouter) Configure(
	userService userservice.Service,
) error {
	userHandlers := userhandlers.NewUserHandlers(userService, r.logger)
	if err := r.RegisterHandlers(context.Background(), userHandlers); err != nil {
		return fmt.Errorf("failed to register handlers: %w", err)
	}
	return nil
}

func (r *UserRouter) RegisterHandlers(
	ctx context.Context,
	handlers userhandlers.Handlers,
) error {
	r.logger.Info("Entering RegisterHandlers")

	// Define the mapping of event topics to handler functions
	eventsToHandlers := map[string]message.NoPublishHandlerFunc{
		userevents.UserSignupRequest:  handlers.HandleUserSignupRequest,
		userevents.UserCreated:        handlers.HandleUserCreated,
		userevents.UserCreationFailed: handlers.HandleUserCreationFailed,
	}

	// Register each handler in the router
	for topic, handlerFunc := range eventsToHandlers {
		handlerName := fmt.Sprintf("user.module.handle.%s.%s", topic, watermill.NewUUID())
		r.logger.Info("Registering handler with AddNoPublisherHandler", slog.String("topic", topic), slog.String("handler", handlerName))

		// Add the handler
		r.router.AddNoPublisherHandler(
			handlerName,  // Handler name
			topic,        // Topic
			r.subscriber, // Subscriber (EventBus)
			handlerFunc,  // Handler function
		)

		r.logger.Info("Handler registered successfully", slog.String("handler", handlerName), slog.String("topic", topic))
	}

	r.logger.Info("Exiting RegisterHandlers")
	return nil
}

// Close stops the router and cleans up resources.
func (r *UserRouter) Close() error {
	return nil
}
