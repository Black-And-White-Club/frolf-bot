package usersubscribers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Black-And-White-Club/tcr-bot/app/events"
	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	user "github.com/Black-And-White-Club/tcr-bot/app/modules/user/interfaces"
	"github.com/Black-And-White-Club/tcr-bot/app/types"
)

// SubscribeToUserEvents subscribes to user-related events using the EventBus.
func SubscribeToUserEvents(ctx context.Context, eventBus events.EventBus, handlers user.Handlers, logger types.LoggerAdapter) error {
	// Subscriber for UserSignupRequest (moved from message_handlers.go)
	if err := eventBus.Subscribe(ctx, userevents.UserSignupRequest.String(), func(ctx context.Context, msg types.Message) error {
		var signupReq userevents.UserSignupRequestPayload
		if err := json.Unmarshal(msg.Payload(), &signupReq); err != nil {
			// Handle the error (e.g., log it)
			return fmt.Errorf("failed to unmarshal UserSignupRequest: %w", err)
		}

		// Call the appropriate handler function (if needed)
		if err := handlers.HandleUserSignupRequest(ctx, msg); err != nil {
			// Handle the error (e.g., log it)
			return fmt.Errorf("failed to handle UserSignupRequest: %w", err)
		}

		msg.Ack()
		return nil
	}); err != nil {
		return fmt.Errorf("failed to subscribe to UserSignupRequest: %w", err)
	}

	// Existing subscriptions
	eventSubscriptions := []struct {
		eventType events.EventType
		handler   func(context.Context, types.Message) error
	}{
		{
			eventType: userevents.UserRoleUpdateRequest,
			handler:   handlers.HandleUserRoleUpdateRequest,
		},
	}

	for _, event := range eventSubscriptions {
		logger.Info("Subscribing to event", types.LogFields{"event_type": event.eventType.String()})
		if err := eventBus.Subscribe(ctx, event.eventType.String(), event.handler); err != nil {
			logger.Error("Failed to subscribe to event", err, types.LogFields{"event_type": event.eventType.String()})
			return fmt.Errorf("failed to subscribe to event %s: %w", event.eventType.String(), err)
		}
		logger.Info("Successfully subscribed to event", types.LogFields{"event_type": event.eventType.String()})
	}

	return nil
}
