package roundrouter

import (
	"context"
	"fmt"
	"log/slog"

	roundservice "github.com/Black-And-White-Club/tcr-bot/app/modules/round/application"
	roundevents "github.com/Black-And-White-Club/tcr-bot/app/modules/round/domain/events"
	roundhandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/round/infrastructure/handlers"
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
	if err := r.RegisterHandlers(context.Background(), roundHandlers, roundService); err != nil {
		return fmt.Errorf("failed to register handlers: %w", err)
	}
	return nil
}

// RegisterHandlers registers the event handlers for the round module.
func (r *RoundRouter) RegisterHandlers(
	ctx context.Context,
	handlers *roundhandlers.RoundHandlers, // Use a pointer to RoundHandlers
	roundService roundservice.Service,
) error {
	r.logger.Info("Entering RegisterHandlers for Round")

	// Define the mapping of event topics to handler functions.
	eventsToHandlers := map[string]message.NoPublishHandlerFunc{
		// Create Round
		roundevents.RoundCreateRequest:  handlers.HandleRoundCreateRequest,
		roundevents.RoundValidated:      handlers.HandleRoundValidated,
		roundevents.RoundDateTimeParsed: handlers.HandleRoundDateTimeParsed,
		roundevents.RoundEntityCreated:  handlers.HandleRoundEntityCreated,
		roundevents.RoundStored:         handlers.HandleRoundStored,
		roundevents.RoundScheduled:      handlers.HandleRoundScheduled,

		// Update Round
		roundevents.RoundUpdateRequest:   handlers.HandleRoundUpdateRequest,
		roundevents.RoundUpdateValidated: handlers.HandleRoundUpdateValidated,
		roundevents.RoundFetched:         handlers.HandleRoundFetched,
		roundevents.RoundEntityUpdated:   handlers.HandleRoundEntityUpdated,
		roundevents.RoundUpdated:         handlers.HandleRoundUpdated,

		// Delete Round
		roundevents.RoundDeleteRequest:       handlers.HandleRoundDeleteRequest,
		roundevents.RoundDeleteValidated:     handlers.HandleRoundDeleteValidated,
		roundevents.RoundToDeleteFetched:     handlers.HandleRoundToDeleteFetched,
		roundevents.RoundDeleteAuthorized:    handlers.HandleRoundDeleteAuthorized,
		roundevents.RoundUserRoleCheckResult: handlers.HandleRoundUserRoleCheckResult,

		// Join Round
		roundevents.RoundParticipantJoinRequest:   handlers.HandleRoundParticipantJoinRequest,
		roundevents.RoundParticipantJoinValidated: handlers.HandleRoundParticipantJoinValidated,
		roundevents.RoundTagNumberFound:           handlers.HandleRoundTagNumberFound,
		roundevents.RoundTagNumberNotFound:        handlers.HandleRoundTagNumberNotFound,

		// Score Round
		roundevents.RoundScoreUpdateRequest:      handlers.HandleRoundScoreUpdateRequest,
		roundevents.RoundScoreUpdateValidated:    handlers.HandleRoundScoreUpdateValidated,
		roundevents.RoundParticipantScoreUpdated: handlers.HandleRoundParticipantScoreUpdated,
		roundevents.RoundAllScoresSubmitted:      handlers.HandleRoundAllScoresSubmitted,
		roundevents.RoundFinalized:               handlers.HandleRoundFinalized,

		// Tag Retrieval
		roundevents.RoundTagNumberRequest:           handlers.HandleRoundTagNumberRequest,
		roundevents.LeaderboardGetTagNumberResponse: handlers.HandleLeaderboardGetTagNumberResponse,
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
