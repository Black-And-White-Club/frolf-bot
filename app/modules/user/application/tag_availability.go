package userservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	usertypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// CheckTagAvailability is a service function that publishes a TagAvailabilityCheckRequested event to the Leaderboard service.
func (s *UserServiceImpl) CheckTagAvailability(ctx context.Context, msg *message.Message, tagNumber int, discordID usertypes.DiscordID) error {
	// Get correlationID from message metadata
	correlationID := msg.Metadata.Get(middleware.CorrelationIDMetadataKey)

	s.logger.Info("Checking tag availability",
		slog.Int("tag_number", tagNumber),
		slog.String("correlation_id", correlationID),
	)

	// Prepare the event payload for the leaderboard tag availability check request
	eventPayload := &userevents.TagAvailabilityCheckRequestedPayload{
		TagNumber: tagNumber,
		DiscordID: discordID, // Pass the Discord ID in the payload
	}

	// Marshal the payload to JSON
	payloadBytes, err := json.Marshal(eventPayload)
	if err != nil {
		s.logger.Error("Failed to marshal LeaderboardTagAvailabilityCheckRequestPayload",
			slog.Any("error", err),
			slog.String("correlation_id", correlationID),
		)
		return fmt.Errorf("failed to marshal payload for tag availability check: %w", err)
	}

	// Create a new message with the payload
	newMessage := message.NewMessage(watermill.NewUUID(), payloadBytes)

	// Set the correlation ID for the new message
	newMessage.Metadata.Set(middleware.CorrelationIDMetadataKey, correlationID)

	// Publish the event to the leaderboard stream
	if err := s.eventBus.Publish(userevents.LeaderboardTagAvailabilityCheckRequest, newMessage); err != nil {
		slog.Error("Failed to publish LeaderboardTagAvailabilityCheckRequest event",
			slog.Any("error", err),
			slog.String("correlation_id", correlationID),
		)
		return fmt.Errorf("failed to publish tag availability check request: %w", err)
	}

	s.logger.Info("Published LeaderboardTagAvailabilityCheckRequest event",
		slog.String("correlation_id", correlationID),
		slog.Int("tag_number", tagNumber),
	)

	return nil
}

// TagUnavailable is a service function that handles the logic when a tag is unavailable.
func (s *UserServiceImpl) TagUnavailable(ctx context.Context, msg *message.Message, tagNumber int, discordID usertypes.DiscordID) error {
	// Handle the case where the tag is not available
	s.logger.Info("Tag is not available", "tag", tagNumber)
	if err := s.PublishUserCreationFailed(ctx, msg, discordID, &tagNumber, "tag not available"); err != nil {
		return fmt.Errorf("failed to publish UserCreationFailed: %w", err)
	}
	return nil
}
