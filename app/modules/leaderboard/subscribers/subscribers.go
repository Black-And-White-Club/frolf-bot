package leaderboardsubscribers

import (
	"context"
	"fmt"

	leaderboardevents "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/events"
	leaderboardhandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/handlers"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// LeaderboardSubscribers subscribes to leaderboard-related events.
type LeaderboardSubscribers struct {
	Subscriber message.Subscriber
	logger     watermill.LoggerAdapter
	Handlers   *leaderboardhandlers.LeaderboardHandlers
}

// NewLeaderboardSubscribers creates a new LeaderboardSubscribers instance.
func NewLeaderboardSubscribers(subscriber message.Subscriber, logger watermill.LoggerAdapter, handlers *leaderboardhandlers.LeaderboardHandlers) *LeaderboardSubscribers {
	return &LeaderboardSubscribers{
		Subscriber: subscriber,
		logger:     logger,
		Handlers:   handlers,
	}
}

// SubscribeToLeaderboardEvents subscribes to leaderboard-related events and routes them to handlers.
func (s *LeaderboardSubscribers) SubscribeToLeaderboardEvents(ctx context.Context) error {
	eventSubscriptions := []struct {
		subject string
		handler func(context.Context, *message.Message) error
	}{
		{
			subject: leaderboardevents.LeaderboardUpdateEventSubject,
			handler: s.Handlers.HandleLeaderboardUpdate,
		},
		{
			subject: leaderboardevents.TagAssignedSubject,
			handler: s.Handlers.HandleTagAssigned,
		},
		{
			subject: leaderboardevents.TagSwapRequestSubject,
			handler: s.Handlers.HandleTagSwapRequest,
		},
		{
			subject: leaderboardevents.GetLeaderboardRequestSubject,
			handler: s.Handlers.HandleGetLeaderboardRequest,
		},
		{
			subject: leaderboardevents.GetTagByDiscordIDRequestSubject,
			handler: s.Handlers.HandleGetTagByDiscordIDRequest,
		},
		{
			subject: leaderboardevents.CheckTagAvailabilityRequestSubject,
			handler: s.Handlers.HandleCheckTagAvailabilityRequest,
		},
	}

	for _, event := range eventSubscriptions {
		msgs, err := s.Subscriber.Subscribe(ctx, event.subject)
		if err != nil {
			s.logger.Error("Failed to subscribe to events", err, watermill.LogFields{
				"subject": event.subject,
			})
			return fmt.Errorf("failed to subscribe to %s events: %w", event.subject, err)
		}
		s.logger.Info("Successfully subscribed to events", watermill.LogFields{
			"subject": event.subject,
		})
		fmt.Printf("[DEBUG] leaderboardsubscribers: Subscribed to subject: %s\n", event.subject)

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
			if !ok {
				logger.Info("Messages channel closed, exiting message processing", nil)
				return
			}

			logger.Info("Received event", watermill.LogFields{
				"message_id": msg.UUID,
				"payload":    string(msg.Payload),
			})

			err := handler(ctx, msg)
			if err != nil {
				logger.Error("Error handling event", err, watermill.LogFields{
					"message_id": msg.UUID,
				})
				// Only Nack if you want the message to be retried
				// msg.Nack()
				// Consider just logging the error and acknowledging the message
				msg.Ack()
				continue
			}

			logger.Info("Event handled successfully", watermill.LogFields{
				"message_id": msg.UUID,
			})
			msg.Ack()
		}
	}
}
