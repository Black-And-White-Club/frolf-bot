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
		return fmt.Errorf("failed to marshal event payload: %w", err)
	}

	msg = message.NewMessage(watermill.NewUUID(), payloadBytes)
	msg.Metadata.Set("correlation_id", correlationID)

	if err := s.eventBus.Publish(ctx, userevents.LeaderboardTagAvailabilityCheckRequest, msg); err != nil {
		return fmt.Errorf("failed to publish CheckTagAvailabilityRequest event: %w", err)
	}

	s.logger.Info("Published CheckTagAvailabilityRequest event", slog.String("correlation_id", correlationID))
	return nil
}
