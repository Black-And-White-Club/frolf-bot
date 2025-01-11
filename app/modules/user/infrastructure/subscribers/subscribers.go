package usersubscribers

import (
	"context"
	"fmt"
	"log/slog"

	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	userstream "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/stream"

	user "github.com/Black-And-White-Club/tcr-bot/app/modules/user/interfaces"
	"github.com/Black-And-White-Club/tcr-bot/app/shared"
	"github.com/ThreeDotsLabs/watermill/message"
)

type UserSubscribers struct {
	eventBus shared.EventBus
	handlers user.Handlers
	logger   *slog.Logger
}

// NewSubscribers creates a new subscribers instance.
func NewSubscribers(eventBus shared.EventBus, handlers user.Handlers, logger *slog.Logger) UserEventSubscriber {
	return &UserSubscribers{
		eventBus: eventBus,
		handlers: handlers,
		logger:   logger,
	}
}

// SubscribeToUserEvents subscribes to user-related events using the EventBus.
func (s *UserSubscribers) SubscribeToUserEvents(ctx context.Context, eventBus shared.EventBus, handlers user.Handlers, logger *slog.Logger) error {

	logger.Debug("Subscribing to UserSignupRequest")
	if err := eventBus.Subscribe(ctx, userstream.UserSignupRequestStreamName, userevents.UserSignupRequest, func(ctx context.Context, msg *message.Message) error {
		if err := handlers.HandleUserSignupRequest(ctx, msg); err != nil {
			return fmt.Errorf("failed to handle UserSignupRequest: %w", err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to subscribe to UserSignupRequest: %w", err)
	}

	logger.Debug("Subscribing to UserRoleUpdateRequest")
	if err := eventBus.Subscribe(ctx, userstream.UserRoleUpdateRequestStreamName, userevents.UserRoleUpdateRequest, func(ctx context.Context, msg *message.Message) error {
		if err := handlers.HandleUserRoleUpdateRequest(ctx, msg); err != nil {
			return fmt.Errorf("failed to handle UserRoleUpdateRequest: %w", err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to subscribe to UserRoleUpdateRequest: %w", err)
	}

	logger.Debug("Subscribing to UserSignupResponse")
	if err := eventBus.Subscribe(ctx, userstream.UserSignupResponseStreamName, userevents.UserSignupResponse, func(ctx context.Context, msg *message.Message) error {
		logger.Info("Received UserSignupResponse", slog.String("payload", string(msg.Payload)))
		msg.Ack()
		return nil
	}); err != nil {
		return fmt.Errorf("failed to subscribe to UserSignupResponse: %w", err)
	}

	return nil
}
