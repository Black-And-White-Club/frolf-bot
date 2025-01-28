package leaderboardservice

import (
	"context"
	"fmt"
	"log/slog"

	leaderboardevents "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/domain/events"
	leaderboardtypes "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/domain/types"
	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	usertypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types"
	"github.com/Black-And-White-Club/tcr-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/google/uuid"
)

// TagAvailabilityCheckRequested handles the TagAvailabilityCheckRequested event.
func (s *LeaderboardService) TagAvailabilityCheckRequested(ctx context.Context, msg *message.Message) error {
	correlationID, eventPayload, err := eventutil.UnmarshalPayload[leaderboardevents.TagAvailabilityCheckRequestedPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal TagAvailabilityCheckRequestedPayload: %w", err)
	}

	s.logger.Info("Handling TagAvailabilityCheckRequested event",
		"correlation_id", correlationID,
		"discord_id", eventPayload.DiscordID,
		"tag_number", eventPayload.TagNumber,
	)

	// Get the active leaderboard
	activeLeaderboard, err := s.LeaderboardDB.GetActiveLeaderboard(ctx)
	if err != nil {
		s.logger.Error("Failed to get active leaderboard", "error", err, "correlation_id", correlationID)
		return fmt.Errorf("failed to get active leaderboard: %w", err)
	}

	s.logger.Info("Active Leaderboard Data", slog.Any("leaderboard_data", activeLeaderboard.LeaderboardData)) // Log the leaderboard data

	// Check tag availability.
	isAvailable, err := s.LeaderboardDB.CheckTagAvailability(ctx, eventPayload.TagNumber)

	if err != nil {
		s.logger.Error("Failed to check tag availability", "error", err, "correlation_id", correlationID)
		return fmt.Errorf("failed to check tag availability: %w", err)
	}

	s.logger.Info("Result of CheckTagAvailability", slog.Bool("is_available", isAvailable)) // Log the result

	// If tag is available, publish TagAssignmentRequested.
	if isAvailable { // This branch should be taken when the tag is available
		assignmentID := uuid.NewString() // Generate a unique ID for the assignment
		s.logger.Info("Tag is available, publishing TagAssignmentRequested event",
			"correlation_id", correlationID,
			"tag_number", eventPayload.TagNumber,
			"assignment_id", assignmentID,
		)

		if err := s.publishTagAssignmentRequested(ctx, msg, eventPayload.DiscordID, eventPayload.TagNumber, assignmentID); err != nil {
			return fmt.Errorf("failed to publish TagAssignmentRequested event: %w", err)
		}
	} else { // This branch should be taken when the tag is NOT available
		// Publish TagUnavailable to notify User module.
		s.logger.Info("Tag is not available, publishing TagUnavailable event",
			"correlation_id", correlationID,
			"tag_number", eventPayload.TagNumber,
		)

		if err := s.publishTagUnavailable(ctx, msg, eventPayload.TagNumber, eventPayload.DiscordID, "Tag is already assigned"); err != nil {
			return fmt.Errorf("failed to publish TagUnavailable event: %w", err)
		}
	}

	return nil
}

// publishTagAssigned publishes a TagAssigned event.
func (s *LeaderboardService) publishTagAssigned(_ context.Context, msg *message.Message, tagNumber int, discordID leaderboardtypes.DiscordID, assignmentID string) error {
	eventPayload := &leaderboardevents.TagAssignedPayload{
		DiscordID:    discordID,
		TagNumber:    tagNumber,
		AssignmentID: assignmentID,
	}

	// Publish the event to the LeaderboardStreamName.
	return s.publishEvent(msg, leaderboardevents.TagAssigned, eventPayload)
}

// publishTagAvailable publishes a TagAvailable event to the User module.
func (s *LeaderboardService) PublishTagAvailable(ctx context.Context, msg *message.Message, payload *leaderboardevents.TagAssignedPayload) error {
	// Construct the payload for the user.tag.available event
	eventPayload := &userevents.TagAvailablePayload{
		DiscordID: usertypes.DiscordID(payload.DiscordID),
		TagNumber: payload.TagNumber,
	}

	// Publish the new message to the user.tag.available topic
	if err := s.publishEvent(msg, userevents.TagAvailable, eventPayload); err != nil {
		s.logger.Error("Failed to publish TagAvailable event to user stream",
			"error", err,
			"correlation_id", msg.Metadata.Get(middleware.CorrelationIDMetadataKey),
		)
		return fmt.Errorf("failed to publish TagAvailable event to user stream: %w", err)
	}

	s.logger.Info("Successfully republished TagAvailable event to user stream",
		"correlation_id", msg.Metadata.Get(middleware.CorrelationIDMetadataKey),
	)

	return nil
}
