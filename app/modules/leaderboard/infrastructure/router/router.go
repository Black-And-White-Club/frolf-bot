package leaderboardrouter

import (
	"context"
	"fmt"
	"log/slog"

	leaderboardservice "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/application"
	leaderboardevents "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/domain/events"
	leaderboardhandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/infrastructure/handlers"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// LeaderboardRouter handles routing for leaderboard module events.
type LeaderboardRouter struct {
	logger     *slog.Logger
	router     *message.Router
	subscriber message.Subscriber
}

// NewLeaderboardRouter creates a new LeaderboardRouter.
func NewLeaderboardRouter(logger *slog.Logger, router *message.Router, subscriber message.Subscriber) *LeaderboardRouter {
	return &LeaderboardRouter{
		logger:     logger,
		router:     router,
		subscriber: subscriber,
	}
}

// Configure sets up the router with the necessary handlers and dependencies.
func (r *LeaderboardRouter) Configure(
	leaderboardService leaderboardservice.Service,
) error {
	leaderboardHandlers := leaderboardhandlers.NewLeaderboardHandlers(leaderboardService, r.logger)
	if err := r.RegisterHandlers(context.Background(), leaderboardHandlers, leaderboardService); err != nil {
		return fmt.Errorf("failed to register handlers: %w", err)
	}
	return nil
}

// RegisterHandlers registers the event handlers for the leaderboard module.
func (r *LeaderboardRouter) RegisterHandlers(
	ctx context.Context,
	handlers *leaderboardhandlers.LeaderboardHandlers, // Use a pointer to LeaderboardHandlers
	leaderboardService leaderboardservice.Service,
) error {
	r.logger.Info("Entering RegisterHandlers for Leaderboard")

	// Define the mapping of event topics to handler functions.
	eventsToHandlers := map[string]message.NoPublishHandlerFunc{
		leaderboardevents.RoundFinalized:                    handlers.HandleRoundFinalized,
		leaderboardevents.LeaderboardUpdateRequested:        handlers.HandleLeaderboardUpdateRequested,
		leaderboardevents.TagSwapRequested:                  handlers.HandleTagSwapRequested,
		leaderboardevents.TagSwapInitiated:                  handlers.HandleTagSwapInitiated,
		leaderboardevents.GetLeaderboardRequest:             handlers.HandleGetLeaderboardRequest,
		leaderboardevents.GetTagByDiscordIDRequest:          handlers.HandleGetTagByDiscordIDRequest,
		leaderboardevents.TagAssigned:                       handlers.HandleTagAssigned,
		leaderboardevents.TagAvailabilityCheckRequest:       handlers.HandleTagAvailabilityCheckRequested,
		leaderboardevents.LeaderboardTagAssignmentRequested: handlers.HandleTagAssignmentRequested, // Add this line
	}
	fmt.Println("eventsToHandlers:", eventsToHandlers) // Print the map contents
	// Register each handler in the router.
	for topic, handlerFunc := range eventsToHandlers {
		handlerName := fmt.Sprintf("leaderboard.module.handle.%s.%s", topic, watermill.NewUUID())
		r.logger.Info("Registering handler with AddNoPublisherHandler", slog.String("topic", topic), slog.String("handler", handlerName))

		// Add the handler.
		r.router.AddNoPublisherHandler(
			handlerName,  // Handler name
			topic,        // Topic
			r.subscriber, // Subscriber (EventBus)
			handlerFunc,  // Handler function
		)

		r.logger.Info("Handler registered successfully", slog.String("handler", handlerName), slog.String("topic", topic))
	}

	r.logger.Info("Exiting RegisterHandlers for Leaderboard")
	return nil
}

// Close stops the router and cleans up resources.
func (r *LeaderboardRouter) Close() error {
	// Currently, no specific resources to close in this router.
	// If there were any, they would be handled here.
	return nil
}
