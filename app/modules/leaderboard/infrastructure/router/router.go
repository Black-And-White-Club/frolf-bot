package leaderboardrouter

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	leaderboardhandlers "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/handlers"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// LeaderboardRouter handles routing for leaderboard module events.
type LeaderboardRouter struct {
	logger           *slog.Logger
	Router           *message.Router
	subscriber       eventbus.EventBus
	helper           utils.Helpers
	middlewareHelper utils.MiddlewareHelpers
}

// NewLeaderboardRouter creates a new LeaderboardRouter.
func NewLeaderboardRouter(logger *slog.Logger, router *message.Router, subscriber eventbus.EventBus, helper utils.Helpers) *LeaderboardRouter {
	return &LeaderboardRouter{
		logger:           logger,
		Router:           router,
		subscriber:       subscriber,
		helper:           helper,
		middlewareHelper: utils.NewMiddlewareHelper(),
	}
}

// Configure sets up the router with the necessary handlers and dependencies.
func (r *LeaderboardRouter) Configure(leaderboardService leaderboardservice.Service) error {
	leaderboardHandlers := leaderboardhandlers.NewLeaderboardHandlers(leaderboardService, r.logger)

	r.Router.AddMiddleware(
		middleware.CorrelationID,
		r.middlewareHelper.CommonMetadataMiddleware("leaderboard"),
		r.middlewareHelper.DiscordMetadataMiddleware(),
		r.middlewareHelper.RoutingMetadataMiddleware(),
		middleware.Recoverer,
		middleware.Retry{MaxRetries: 3}.Middleware,
	)

	if err := r.RegisterHandlers(context.Background(), leaderboardHandlers); err != nil {
		return fmt.Errorf("failed to register handlers: %w", err)
	}
	return nil
}

// RegisterHandlers registers the event handlers for the leaderboard module.
func (r *LeaderboardRouter) RegisterHandlers(ctx context.Context, handlers leaderboardhandlers.Handlers) error {
	r.logger.Info("🚀 Entering RegisterHandlers for Leaderboard")

	eventsToHandlers := map[string]message.NoPublishHandlerFunc{
		leaderboardevents.RoundFinalized:                    handlers.HandleRoundFinalized,
		leaderboardevents.LeaderboardUpdateRequested:        handlers.HandleLeaderboardUpdateRequested,
		leaderboardevents.TagSwapRequested:                  handlers.HandleTagSwapRequested,
		leaderboardevents.TagSwapInitiated:                  handlers.HandleTagSwapInitiated,
		leaderboardevents.GetLeaderboardRequest:             handlers.HandleGetLeaderboardRequest,
		leaderboardevents.GetTagByUserIDRequest:             handlers.HandleGetTagByUserIDRequest,
		leaderboardevents.TagAssigned:                       handlers.HandleTagAssigned,
		leaderboardevents.TagAvailabilityCheckRequest:       handlers.HandleTagAvailabilityCheckRequested,
		leaderboardevents.LeaderboardTagAssignmentRequested: handlers.HandleTagAssignmentRequested,
	}

	for topic, handlerFunc := range eventsToHandlers {
		handlerName := fmt.Sprintf("leaderboard.%s", topic)

		r.logger.Info("✅ Registering leaderboard handler",
			slog.String("topic", topic),
			slog.String("handler", handlerName),
		)

		r.Router.AddNoPublisherHandler(
			handlerName,
			topic,
			r.subscriber,
			handlerFunc,
		)
	}
	return nil
}
