package userservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// CheckTagAvailability publishes a CheckTagAvailabilityRequest event to the Leaderboard service.
func (s *UserServiceImpl) CheckTagAvailability(ctx context.Context, msg *message.Message, tagNumber int) error {
	correlationID := msg.Metadata.Get(middleware.CorrelationIDMetadataKey)
	s.logger.Info("Requesting tag availability check",
		slog.Int("tag_number", tagNumber),
		slog.String("correlation_id", correlationID),
	)

	payloadBytes, err := json.Marshal(userevents.CheckTagAvailabilityRequestPayload{
		TagNumber: tagNumber,
	})
	if err != nil {
		s.logger.Error("Failed to marshal event payload",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to marshal event payload: %w", err)
	}

	// Create a new message for the outgoing event
	newMsg := message.NewMessage(watermill.NewUUID(), payloadBytes)

	// Use eventutil.PropagateMetadata to copy the correlation ID
	s.eventUtil.PropagateMetadata(msg, newMsg)

	// Set the context on the new message
	newMsg.SetContext(ctx)

	// Publish the new message using s.eventBus.Publish
	if err := s.eventBus.Publish(userevents.LeaderboardTagAvailabilityCheckRequest, newMsg); err != nil {
		s.logger.Error("Failed to publish CheckTagAvailabilityRequest event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to publish CheckTagAvailabilityRequest event: %w", err)
	}

	s.logger.Info("Published CheckTagAvailabilityRequest event", slog.String("correlation_id", correlationID))
	return nil
}
