package roundservice

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"

	"log/slog"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/utils"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// RoundService handles round-related logic.
type RoundService struct {
	RoundDB        rounddb.RoundDB
	EventBus       eventbus.EventBus
	logger         observability.Logger
	roundValidator roundutil.RoundValidator
	metrics        observability.Metrics
	tracer         observability.Tracer
}

// NewRoundService creates a new RoundService.
func NewRoundService(db rounddb.RoundDB, eventBus eventbus.EventBus, logger observability.Logger, metrics observability.Metrics, tracer observability.Tracer) Service {
	return &RoundService{
		RoundDB:        db,
		EventBus:       eventBus,
		logger:         logger,
		roundValidator: roundutil.NewRoundValidator(),
		metrics:        metrics,
		tracer:         tracer,
	}
}

// publishEvent is a generic helper function to publish events.
func (s *RoundService) publishEvent(msg *message.Message, eventName string, payload interface{}) error {
	// Extract correlation ID from incoming message
	correlationID := msg.Metadata.Get(middleware.CorrelationIDMetadataKey)
	if correlationID == "" {
		correlationID = watermill.NewUUID()
	}

	// Create a new message with the payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload for event %s: %w", eventName, err)
	}

	// Create new message with proper metadata
	newMessage := message.NewMessage(watermill.NewUUID(), payloadBytes)

	// Transfer essential metadata
	newMessage.Metadata.Set(middleware.CorrelationIDMetadataKey, correlationID)

	// Add context about the origin
	callerFunc := getCallerFunctionName()
	if callerFunc != "" {
		newMessage.Metadata.Set("caused_by", callerFunc)
	}

	// Add any additional relevant metadata from the original message
	if parentID := msg.Metadata.Get("parent_id"); parentID != "" {
		newMessage.Metadata.Set("parent_id", parentID)
	}

	// Log the intention to publish at debug level
	s.logger.Debug("Publishing event",
		slog.String("event", eventName),
		slog.String("correlation_id", correlationID),
		slog.String("message_id", newMessage.UUID),
	)

	// Publish the event
	if err := s.EventBus.Publish(eventName, newMessage); err != nil {
		s.logger.Error("Failed to publish event",
			slog.String("event", eventName),
			slog.String("correlation_id", correlationID),
			slog.String("message_id", newMessage.UUID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to publish event %s: %w", eventName, err)
	}

	// Log success at debug level only
	s.logger.Debug("Event published",
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

func (s *RoundService) getEventMessageID(ctx context.Context, roundID roundtypes.ID) (roundtypes.EventMessageID, error) {
	eventMessageID, err := s.RoundDB.GetEventMessageID(ctx, roundID)
	slog.Info("We are here 🌟")
	if err != nil {
		return "", fmt.Errorf("failed to retrieve EventMessageID for round %d: %w", roundID, err)
	}
	return *eventMessageID, nil
}
