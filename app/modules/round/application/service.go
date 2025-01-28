package roundservice

import (
	"encoding/json"
	"fmt"
	"runtime"

	"log/slog"

	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/infrastructure/repositories"
	roundutil "github.com/Black-And-White-Club/tcr-bot/app/modules/round/utils"
	"github.com/Black-And-White-Club/tcr-bot/app/shared"
	"github.com/Black-And-White-Club/tcr-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// RoundService handles round-related logic.
type RoundService struct {
	RoundDB        rounddb.RoundDB
	EventBus       shared.EventBus
	logger         *slog.Logger
	eventUtil      eventutil.EventUtil
	roundValidator roundutil.RoundValidator
}

// NewRoundService creates a new RoundService.
func NewRoundService(db rounddb.RoundDB, eventBus shared.EventBus, logger *slog.Logger) Service {
	return &RoundService{
		RoundDB:        db,
		EventBus:       eventBus,
		logger:         logger,
		eventUtil:      eventutil.NewEventUtil(),
		roundValidator: roundutil.NewRoundValidator(),
	}
}

// publishEvent is a generic helper function to publish events.
func (s *RoundService) publishEvent(msg *message.Message, eventName string, payload interface{}) error {
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

	// Set caused_by metadata to the name of the calling function
	newMessage.Metadata.Set("caused_by", getCallerFunctionName())

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

// getCallerFunctionName is a helper function to get the name of the calling function.
func getCallerFunctionName() string {
	pc, _, _, ok := runtime.Caller(1) // 1 level up the call stack
	if !ok {
		return "unknown"
	}
	return runtime.FuncForPC(pc).Name() // Get the function name
}
