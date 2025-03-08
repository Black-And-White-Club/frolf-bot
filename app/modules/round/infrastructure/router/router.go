package roundrouter

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	roundhandlers "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/handlers"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// RoundRouter handles routing for round module events.
type RoundRouter struct {
	logger           *slog.Logger
	Router           *message.Router
	subscriber       eventbus.EventBus
	helper           utils.Helpers
	middlewareHelper utils.MiddlewareHelpers
}

// NewRoundRouter creates a new RoundRouter.
func NewRoundRouter(logger *slog.Logger, router *message.Router, subscriber eventbus.EventBus, helper utils.Helpers) *RoundRouter {
	return &RoundRouter{
		logger:           logger,
		Router:           router,
		subscriber:       subscriber,
		helper:           helper,
		middlewareHelper: utils.NewMiddlewareHelper(),
	}
}

// Configure sets up the router with the necessary handlers and dependencies.
func (r *RoundRouter) Configure(roundService roundservice.Service) error {
	roundHandlers := roundhandlers.NewRoundHandlers(roundService, r.logger)

	r.Router.AddMiddleware(
		middleware.CorrelationID,
		r.middlewareHelper.CommonMetadataMiddleware("round"),
		r.middlewareHelper.DiscordMetadataMiddleware(),
		r.middlewareHelper.RoutingMetadataMiddleware(),
		middleware.Recoverer,
		middleware.Retry{MaxRetries: 3}.Middleware,
	)

	if err := r.RegisterHandlers(context.Background(), roundHandlers); err != nil {
		return fmt.Errorf("failed to register handlers: %w", err)
	}
	return nil
}

// RegisterHandlers registers the event handlers for the round module.
func (r *RoundRouter) RegisterHandlers(ctx context.Context, handlers roundhandlers.Handlers) error {
	r.logger.Info("Entering RegisterHandlers for Round")

	eventsToHandlers := map[string]message.NoPublishHandlerFunc{
		roundevents.RoundCreateRequest:            handlers.HandleRoundCreateRequest,
		roundevents.RoundStored:                   handlers.HandleRoundStored,
		roundevents.RoundValidated:                handlers.HandleRoundValidated,
		roundevents.RoundEntityCreated:            handlers.HandleRoundEntityCreated,
		roundevents.RoundScheduled:                handlers.HandleRoundScheduled,
		roundevents.RoundDiscordEventIDUpdate:     handlers.HandleUpdateDiscordEventID,
		roundevents.RoundUpdateRequest:            handlers.HandleRoundUpdateRequest,
		roundevents.RoundUpdateValidated:          handlers.HandleRoundUpdateValidated,
		roundevents.RoundFetched:                  handlers.HandleRoundFetched,
		roundevents.RoundEntityUpdated:            handlers.HandleRoundEntityUpdated,
		roundevents.RoundScheduleUpdate:           handlers.HandleRoundScheduleUpdate,
		roundevents.RoundDeleteRequest:            handlers.HandleRoundDeleteRequest,
		roundevents.RoundDeleteValidated:          handlers.HandleRoundDeleteValidated,
		roundevents.RoundToDeleteFetched:          handlers.HandleRoundToDeleteFetched,
		roundevents.RoundDeleteAuthorized:         handlers.HandleRoundDeleteAuthorized,
		roundevents.RoundUserRoleCheckResult:      handlers.HandleRoundUserRoleCheckResult,
		roundevents.RoundStarted:                  handlers.HandleRoundStarted,
		roundevents.RoundReminder:                 handlers.HandleRoundReminder,
		roundevents.RoundParticipantJoinRequest:   handlers.HandleRoundParticipantJoinRequest,
		roundevents.RoundParticipantJoinValidated: handlers.HandleRoundParticipantJoinValidated,
		roundevents.RoundTagNumberFound:           handlers.HandleRoundTagNumberFound,
		roundevents.RoundTagNumberNotFound:        handlers.HandleRoundTagNumberNotFound,
	}

	for topic, handlerFunc := range eventsToHandlers {
		handlerName := fmt.Sprintf("round.%s", topic) // Simplified handler name
		r.Router.AddNoPublisherHandler(
			handlerName,
			topic,
			r.subscriber,
			handlerFunc,
		)
	}

	r.logger.Info("Exiting RegisterHandlers for Round")
	return nil
}

// Close stops the router and cleans up resources.
func (r *RoundRouter) Close() error {
	return nil
}
