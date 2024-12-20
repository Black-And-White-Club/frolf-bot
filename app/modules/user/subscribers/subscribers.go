package usersubscribers

import (
	"context"
	"fmt"

	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/events"
	userhandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/user/handlers"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// SubscribeToUserEvents subscribes to user-related events and routes them to handlers.
func SubscribeToUserEvents(ctx context.Context, subscriber message.Subscriber, handlers userhandlers.Handlers, logger watermill.LoggerAdapter,
) error {
	eventSubscriptions := []struct {
		subject string
		handler func(context.Context, *message.Message) error
	}{
		{
			subject: userevents.UserSignupRequestSubject,
			handler: handlers.HandleUserSignupRequest,
		},
		{
			subject: userevents.UserRoleUpdateRequestSubject,
			handler: handlers.HandleUserRoleUpdateRequest,
		},
	}

	for _, event := range eventSubscriptions {
		logger.Info("Subscribing to events", watermill.LogFields{"subject": event.subject})
		msgs, err := subscriber.Subscribe(ctx, event.subject)
		if err != nil {
			logger.Error("Failed to subscribe to events", err, watermill.LogFields{"subject": event.subject})
			return fmt.Errorf("failed to subscribe to %s events: %w", event.subject, err)
		}
		logger.Info("Successfully subscribed to events", watermill.LogFields{"subject": event.subject})

		go processEventMessages(ctx, msgs, event.handler, logger)
	}

	return nil
}

func processEventMessages(
	ctx context.Context,
	messages <-chan *message.Message,
	handler func(context.Context, *message.Message) error,
	logger watermill.LoggerAdapter,
) {
	logger.Info("Starting message processing", nil)
	defer logger.Info("Exiting message processing", nil)

	for {
		select {
		case <-ctx.Done():
			logger.Info("Context cancelled, exiting message processing", nil)
			return
		case msg, ok := <-messages:
			if !ok {
				logger.Info("Messages channel closed, exiting message processing", nil)
				return
			}

			logger.Info("Received event", watermill.LogFields{
				"message_id": msg.UUID,
				"payload":    string(msg.Payload),
			})

			if err := handler(ctx, msg); err != nil {
				logger.Error("Error handling event", err, watermill.LogFields{
					"message_id": msg.UUID,
				})
				msg.Nack()
				continue
			}

			logger.Info("Event handled successfully", watermill.LogFields{
				"message_id": msg.UUID,
			})
			msg.Ack()
		}
	}
}
