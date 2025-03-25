package leaderboardservice

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// LeaderboardService handles leaderboard logic.
type LeaderboardService struct {
	LeaderboardDB leaderboarddb.LeaderboardDB
	EventBus      eventbus.EventBus
	logger        observability.Logger
	metrics       observability.Metrics
	tracer        observability.Tracer
}

// NewLeaderboardService creates a new LeaderboardService.
func NewLeaderboardService(db leaderboarddb.LeaderboardDB, eventBus eventbus.EventBus, logger observability.Logger, metrics observability.Metrics, tracer observability.Tracer) Service {
	return &LeaderboardService{
		LeaderboardDB: db,
		EventBus:      eventBus,
		logger:        logger,
		metrics:       metrics,
		tracer:        tracer,
	}
}

// publishEvent is a generic helper function to publish events.
func (s *LeaderboardService) publishEvent(msg *message.Message, eventName string, payload interface{}) error {
	correlationID := msg.Metadata.Get(middleware.CorrelationIDMetadataKey)

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		s.logger.Error("Failed to marshal event payload",
			slog.String("event", eventName),
			slog.Any("error", err),
			slog.String("correlation_id", correlationID),
		)
		return fmt.Errorf("failed to marshal event payload for %s: %w", eventName, err)
	}

	newMessage := message.NewMessage(watermill.NewUUID(), payloadBytes)

	// Set Nats-Msg-Id for JetStream deduplication
	newMessage.Metadata.Set("Nats-Msg-Id", newMessage.UUID+"-"+eventName)

	if err := s.EventBus.Publish(eventName, newMessage); err != nil {
		s.logger.Error("Failed to publish event",
			slog.String("event", eventName),
			slog.Any("error", err),
			slog.String("correlation_id", correlationID),
		)
		return fmt.Errorf("failed to publish event %s: %w", eventName, err)
	}

	s.logger.Info("Published event",
		slog.String("event", eventName),
		slog.String("correlation_id", correlationID),
		slog.String("message_id", newMessage.UUID),
	)

	return nil
}
