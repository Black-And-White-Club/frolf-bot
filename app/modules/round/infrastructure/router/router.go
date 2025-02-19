package roundrouter

import (
	"context"
	"fmt"
	"log/slog"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	roundhandlers "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/handlers"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// RoundRouter handles routing for round module events.
type RoundRouter struct {
	logger     *slog.Logger
	router     *message.Router
	subscriber message.Subscriber
}

// NewRoundRouter creates a new RoundRouter.
func NewRoundRouter(logger *slog.Logger, router *message.Router, subscriber message.Subscriber) *RoundRouter {
	return &RoundRouter{
		logger:     logger,
		router:     router,
		subscriber: subscriber,
	}
}

// Configure sets up the router with the necessary handlers and dependencies.
func (r *RoundRouter) Configure(
	roundService roundservice.Service,
) error {
	roundHandlers := roundhandlers.NewRoundHandlers(roundService, r.logger).(*roundhandlers.RoundHandlers)
	if err := r.RegisterHandlers(context.Background(), roundHandlers); err != nil {
		return fmt.Errorf("failed to register handlers: %w", err)
	}
	return nil
}

// RegisterHandlers registers the event handlers for the round module.
func (r *RoundRouter) RegisterHandlers(
	ctx context.Context,
	handlers roundhandlers.Handlers, // Use a pointer to RoundHandlers
) error {
	r.logger.Info("Entering RegisterHandlers for Round")

	// Define the mapping of event topics to handler functions.
	eventsToHandlers := map[string]message.NoPublishHandlerFunc{
		// Create Round
		roundevents.RoundCreateRequest: handlers.HandleRoundCreateRequest,
		roundevents.RoundValidated:     handlers.HandleRoundValidated,
		roundevents.RoundStored:        handlers.HandleRoundStored,
		roundevents.RoundScheduled:     handlers.HandleRoundScheduled,

		// Update Discord Event ID
		roundevents.RoundDiscordEventIDUpdate: handlers.HandleUpdateDiscordEventID,

		// Update Round
		roundevents.RoundUpdateRequest:   handlers.HandleRoundUpdateRequest,
		roundevents.RoundUpdateValidated: handlers.HandleRoundUpdateValidated,
		roundevents.RoundFetched:         handlers.HandleRoundFetched,
		roundevents.RoundEntityUpdated:   handlers.HandleRoundEntityUpdated,
		roundevents.RoundScheduleUpdate:  handlers.HandleRoundScheduleUpdate,

		// Delete Round
		roundevents.RoundDeleteRequest:       handlers.HandleRoundDeleteRequest,
		roundevents.RoundDeleteValidated:     handlers.HandleRoundDeleteValidated,
		roundevents.RoundToDeleteFetched:     handlers.HandleRoundToDeleteFetched,
		roundevents.RoundDeleteAuthorized:    handlers.HandleRoundDeleteAuthorized,
		roundevents.RoundUserRoleCheckResult: handlers.HandleRoundUserRoleCheckResult,

		// Start Round
		roundevents.RoundStarted: handlers.HandleRoundStarted,

		// Remind Round
		roundevents.RoundReminder: handlers.HandleRoundReminder,

		// Join Round
		roundevents.RoundParticipantJoinRequest:   handlers.HandleRoundParticipantJoinRequest,
		roundevents.RoundParticipantJoinValidated: handlers.HandleRoundParticipantJoinValidated,
		roundevents.RoundTagNumberFound:           handlers.HandleRoundTagNumberFound,
		roundevents.RoundTagNumberNotFound:        handlers.HandleRoundTagNumberNotFound,
	}

	// Register each handler in the router.
	for topic, handlerFunc := range eventsToHandlers {
		handlerName := fmt.Sprintf("round.module.handle.%s.%s", topic, watermill.NewUUID())
		r.logger.Info("Registering handler with AddNoPublisherHandle", slog.String("topic", topic), slog.String("handler", handlerName))

		// Add the handler.
		r.router.AddNoPublisherHandler(
			handlerName,  // Handler name
			topic,        // Topic
			r.subscriber, // Subscriber (EventBus)
			handlerFunc,  // Handler function
		)

		r.logger.Info("Handler registered successfully", slog.String("handler", handlerName), slog.String("topic", topic))
	}

	r.logger.Info("Exiting RegisterHandlers for Round")
	return nil
}

// Close stops the router and cleans up resources.
func (r *RoundRouter) Close() error {
	// Currently, no specific resources to close in this router.
	// If there were any, they would be handled here.
	return nil
}
