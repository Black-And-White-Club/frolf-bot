package scorerouter

import (
	"context"
	"fmt"
	"log/slog"

	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	scoreservice "github.com/Black-And-White-Club/frolf-bot/app/modules/score/application"
	scorehandlers "github.com/Black-And-White-Club/frolf-bot/app/modules/score/infrastructure/handlers"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// ScoreRouter handles routing for score module events.
type ScoreRouter struct {
	logger     *slog.Logger
	router     *message.Router
	subscriber message.Subscriber
}

// NewScoreRouter creates a new ScoreRouter.
func NewScoreRouter(logger *slog.Logger, router *message.Router, subscriber message.Subscriber) *ScoreRouter {
	return &ScoreRouter{
		logger:     logger,
		router:     router,
		subscriber: subscriber,
	}
}

// Configure sets up the router with the necessary handlers and dependencies.
func (r *ScoreRouter) Configure(
	scoreService scoreservice.Service,
) error {
	scoreHandlers := scorehandlers.NewScoreHandlers(scoreService, r.logger)
	// Add middleware to enhance logging and deduplication
	r.router.AddMiddleware(
		middleware.Recoverer,
		middleware.CorrelationID,
	)
	if err := r.RegisterHandlers(context.Background(), scoreHandlers); err != nil {
		return fmt.Errorf("failed to register handlers: %w", err)
	}
	return nil
}

// RegisterHandlers registers the event handlers for the score module.
func (r *ScoreRouter) RegisterHandlers(ctx context.Context, handlers scorehandlers.Handlers) error {
	r.logger.Info("Entering RegisterHandlers for Score")

	// Define the mapping of event topics to handler functions.
	eventsToHandlers := map[string]message.NoPublishHandlerFunc{
		scoreevents.ProcessRoundScoresRequest: handlers.HandleProcessRoundScoresRequest,
		scoreevents.ScoreCorrectionRequest:    handlers.HandleScoreUpdateRequest, // Assuming this handles score corrections
	}

	// Register each handler in the router.
	for topic, handlerFunc := range eventsToHandlers {
		handlerName := fmt.Sprintf("score.module.handle.%s.%s", topic, watermill.NewUUID())
		r.logger.Info("Registering handler with AddHandler", slog.String("topic", topic), slog.String("handler", handlerName))

		// Add the handler.
		r.router.AddNoPublisherHandler(
			handlerName,  // Handler name
			topic,        // Topic
			r.subscriber, // Subscriber (EventBus)
			handlerFunc,  // Handler function
		)

		r.logger.Info("Handler registered successfully", slog.String("handler", handlerName), slog.String("topic", topic))
	}

	r.logger.Info("Exiting RegisterHandlers for Score")
	return nil
}

// Close stops the router and cleans up resources.
func (r *ScoreRouter) Close() error {
	// Currently, no specific resources to close in this router.
	// If there were any, they would be handled here.
	return nil
}
