package scoresubscribers

import (
	"context"
	"encoding/json"
	"fmt"

	scoreevents "github.com/Black-And-White-Club/tcr-bot/app/modules/score/events"
	scorehandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/score/handlers"
	scoreservice "github.com/Black-And-White-Club/tcr-bot/app/modules/score/service"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// ScoreSubscribers subscribes to round-related events.
type ScoreSubscribers struct {
	Subscriber message.Subscriber
	logger     watermill.LoggerAdapter
	Handlers   scorehandlers.Handlers
	Service    scoreservice.Service
}

// NewScoreSubscribers creates a new ScoreSubscribers instance.
func NewScoreSubscribers(subscriber message.Subscriber, logger watermill.LoggerAdapter, handlers scorehandlers.Handlers, service scoreservice.Service) *ScoreSubscribers {
	return &ScoreSubscribers{
		Subscriber: subscriber,
		logger:     logger,
		Handlers:   handlers,
		Service:    service,
	}
}

// SubscribeToScoreEvents subscribes to score-related events and routes them to handlers.
func (s *ScoreSubscribers) SubscribeToScoreEvents(ctx context.Context) error {
	eventSubscriptions := []struct {
		subject string
		handler func(context.Context, *message.Message) error
	}{
		{
			subject: scoreevents.ScoresReceivedEventSubject,
			handler: s.handleScoresReceived,
		},
		{
			subject: scoreevents.ScoreCorrectedEventSubject,
			handler: s.Handlers.HandleScoreCorrected,
		},
	}

	for _, event := range eventSubscriptions {
		s.logger.Info("Subscribing to events", watermill.LogFields{"subject": event.subject})
		msgs, err := s.Subscriber.Subscribe(ctx, event.subject)
		if err != nil {
			s.logger.Error("Failed to subscribe to events", err, watermill.LogFields{"subject": event.subject})
			return fmt.Errorf("failed to subscribe to %s events: %w", event.subject, err)
		}
		s.logger.Info("Successfully subscribed to events", watermill.LogFields{"subject": event.subject})

		go processEventMessages(ctx, msgs, event.handler, s.logger)
	}

	return nil
}

func (s *ScoreSubscribers) handleScoresReceived(ctx context.Context, msg *message.Message) error {
	var event scoreevents.ScoresReceivedEvent
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		return fmt.Errorf("failed to unmarshal ScoresReceivedEvent: %w", err)
	}

	if err := s.Service.ProcessRoundScores(ctx, event); err != nil {
		return fmt.Errorf("failed to process round scores: %w", err)
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
