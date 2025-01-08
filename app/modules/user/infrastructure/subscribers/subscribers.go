package usersubscribers

import (
	"context"
	"fmt"

	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	user "github.com/Black-And-White-Club/tcr-bot/app/modules/user/interfaces"
	"github.com/Black-And-White-Club/tcr-bot/app/shared"
)

type UserSubscribers struct {
	eventBus shared.EventBus
	handlers user.Handlers
	logger   shared.LoggerAdapter
}

// NewSubscribers creates a new subscribers instance.
func NewSubscribers(eventBus shared.EventBus, handlers user.Handlers, logger shared.LoggerAdapter) UserEventSubscriber {
	return &UserSubscribers{
		eventBus: eventBus,
		handlers: handlers,
		logger:   logger,
	}
}

// SubscribeToUserEvents subscribes to user-related events using the EventBus.
func (s *UserSubscribers) SubscribeToUserEvents(ctx context.Context, eventBus shared.EventBus, handlers user.Handlers, logger shared.LoggerAdapter) error {

	// Subscribe to UserSignupRequest
	if err := eventBus.Subscribe(ctx, userevents.UserSignupRequest.String(), func(ctx context.Context, msg shared.Message) error {
		if err := handlers.HandleUserSignupRequest(ctx, msg); err != nil {
			return fmt.Errorf("failed to handle UserSignupRequest: %w", err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to subscribe to UserSignupRequest: %w", err)
	}

	// Subscribe to UserRoleUpdateRequest
	if err := eventBus.Subscribe(ctx, userevents.UserRoleUpdateRequest.String(), func(ctx context.Context, msg shared.Message) error {
		if err := handlers.HandleUserRoleUpdateRequest(ctx, msg); err != nil {
			return fmt.Errorf("failed to handle UserRoleUpdateRequest: %w", err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to subscribe to UserRoleUpdateRequest: %w", err)
	}

	return nil
}
