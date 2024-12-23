package usersubscribers

import (
	"context"
	"fmt"

	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/events"
	userhandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/user/handlers"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// UserSubscribers struct holds the subscriber, handlers, and logger.
type UserSubscribers struct {
	subscriber message.Subscriber
	handlers   userhandlers.Handlers
	logger     watermill.LoggerAdapter
}

// Close delegates the Close call to the underlying watermill.Subscriber.
func (s *UserSubscribers) Close() error {
	return s.subscriber.Close() // Delegate to the embedded subscriber
}

// NewUserSubscribers creates a new UserSubscribers instance and returns the UserEventSubscriber and Closer interfaces.
func NewUserSubscribers(subscriber message.Subscriber, handlers userhandlers.Handlers, logger watermill.LoggerAdapter) (UserEventSubscriber, Closer, error) {
	if subscriber == nil {
		return nil, nil, fmt.Errorf("subscriber cannot be nil")
	}

	if handlers == nil {
		return nil, nil, fmt.Errorf("handlers cannot be nil")
	}

	if logger == nil {
		return nil, nil, fmt.Errorf("logger cannot be nil")
	}

	userSubscribers := &UserSubscribers{
		subscriber: subscriber,
		handlers:   handlers,
		logger:     logger,
	}

	return userSubscribers, userSubscribers, nil
}

// SubscribeToUserEvents subscribes to user-related events and routes them to handlers.
func (s *UserSubscribers) SubscribeToUserEvents(ctx context.Context) error {
	eventSubscriptions := []struct {
		subject string
		handler func(context.Context, *message.Message) error
	}{
		{
			subject: userevents.UserSignupRequestSubject,
			handler: s.handlers.HandleUserSignupRequest,
		},
		{
			subject: userevents.UserRoleUpdateRequestSubject,
			handler: s.handlers.HandleUserRoleUpdateRequest,
		},
	}

	for _, event := range eventSubscriptions {
		s.logger.Info("Subscribing to events", watermill.LogFields{"subject": event.subject}) // Log BEFORE subscribing
		msgs, err := s.subscriber.Subscribe(ctx, event.subject)
		if err != nil {
			s.logger.Error("Failed to subscribe to events", err, watermill.LogFields{"subject": event.subject})
			return fmt.Errorf("failed to subscribe to %s events: %w", event.subject, err)
		}
		s.logger.Info("Successfully subscribed to events", watermill.LogFields{"subject": event.subject}) // Log AFTER successful subscription

		go processEventMessages(ctx, msgs, event.handler, s.logger)
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
			fmt.Println("processEventMessages: Received message or channel closed")
			if !ok {
				fmt.Println("processEventMessages: Messages channel closed")
				logger.Info("Messages channel closed, exiting message processing", nil)
				return
			}

			// Check for the "stop" message
			if string(msg.Payload) == "STOP" {
				fmt.Println("processEventMessages: Received STOP message")
				return // Exit the loop
			}

			fmt.Println("processEventMessages: Processing message")
			logger.Info("Received event", watermill.LogFields{
				"message_id": msg.UUID,
				"payload":    string(msg.Payload),
			})

			if err := handler(ctx, msg); err != nil {
				logger.Error("Error handling event", err, watermill.LogFields{
					"message_id": msg.UUID,
				})
				msg.Nack()
				fmt.Println("processEventMessages: Error handling message")
				continue
			}

			fmt.Println("processEventMessages: Message processed")
			logger.Info("Event handled successfully", watermill.LogFields{
				"message_id": msg.UUID,
			})
			msg.Ack()
		}
	}
}
