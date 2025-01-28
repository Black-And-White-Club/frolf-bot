package leaderboardservice

import (
	"encoding/json"
	"fmt"
	"log/slog"

	leaderboarddb "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/Black-And-White-Club/tcr-bot/app/shared"
	"github.com/Black-And-White-Club/tcr-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// LeaderboardService handles leaderboard logic.
type LeaderboardService struct {
	LeaderboardDB leaderboarddb.LeaderboardDB
	EventBus      shared.EventBus
	logger        *slog.Logger
	eventUtil     eventutil.EventUtil
}

// NewLeaderboardService creates a new LeaderboardService.
func NewLeaderboardService(db leaderboarddb.LeaderboardDB, eventBus shared.EventBus, logger *slog.Logger) Service {
	return &LeaderboardService{
		LeaderboardDB: db,
		EventBus:      eventBus,
		logger:        logger,
		eventUtil:     eventutil.NewEventUtil(),
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
	s.eventUtil.PropagateMetadata(msg, newMessage)

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
