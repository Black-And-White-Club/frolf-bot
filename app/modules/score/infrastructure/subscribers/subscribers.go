package scoresubscribers

import (
	"context"
	"fmt"
	"log/slog"

	scoreevents "github.com/Black-And-White-Club/tcr-bot/app/modules/score/domain/events"
	scorehandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/score/infrastructure/handlers"
	"github.com/Black-And-White-Club/tcr-bot/app/shared"
	"github.com/ThreeDotsLabs/watermill/message"
)

// ScoreSubscribers subscribes to score-related events.
type ScoreSubscribers struct {
	eventBus shared.EventBus
	handlers scorehandlers.Handlers
	logger   *slog.Logger
}

// NewScoreSubscribers creates a new ScoreSubscribers instance.
func NewSubscribers(eventBus shared.EventBus, handlers scorehandlers.Handlers, logger *slog.Logger) Subscribers {
	return &ScoreSubscribers{
		eventBus: eventBus,
		handlers: handlers,
		logger:   logger,
	}
}

// SubscribeToScoreEvents subscribes to score-related events using the EventBus.
func (s *ScoreSubscribers) SubscribeToScoreEvents(ctx context.Context) error {
	s.logger.Debug("Subscribing to ScoresReceivedEvent")
	if err := s.eventBus.Subscribe(ctx, scoreevents.ScoreStreamName, scoreevents.ScoresReceivedEventSubject, func(ctx context.Context, msg *message.Message) error {
		s.logger.Info("ScoresReceivedEvent handler invoked")
		s.logger.Debug("Message received", slog.Any("msg", msg))

		if err := s.handlers.HandleScoresReceived(ctx, msg); err != nil {
			s.logger.Error("Failed to handle ScoresReceivedEvent", "error", err) // Log the error
		}
		return nil // Do not return the error
	}); err != nil {
		return fmt.Errorf("failed to subscribe to ScoresReceivedEvent: %w", err)
	}

	s.logger.Debug("Subscribing to ScoreCorrectedEvent")
	if err := s.eventBus.Subscribe(ctx, scoreevents.ScoreStreamName, scoreevents.ScoreCorrectedEventSubject, func(ctx context.Context, msg *message.Message) error {
		if err := s.handlers.HandleScoreCorrected(ctx, msg); err != nil {
			s.logger.Error("Failed to handle ScoreCorrectedEvent", "error", err) // Log the error
		}
		return nil // Do not return the error
	}); err != nil {
		return fmt.Errorf("failed to subscribe to ScoreCorrectedEvent: %w", err)
	}

	s.logger.Debug("Subscribing to ProcessedScoresEvent")
	if err := s.eventBus.Subscribe(ctx, scoreevents.ScoreStreamName, scoreevents.ProcessedScoresEventSubject, func(ctx context.Context, msg *message.Message) error {
		s.logger.Info("Received ProcessedScoresEvent", slog.String("payload", string(msg.Payload)))
		msg.Ack()
		return nil
	}); err != nil {
		return fmt.Errorf("failed to subscribe to ProcessedScoresEvent: %w", err)
	}

	return nil
}
