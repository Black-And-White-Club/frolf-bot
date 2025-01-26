package leaderboardservice

import (
	"context"
	"fmt"

	leaderboardevents "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/domain/events"
	leaderboardtypes "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/domain/types"
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

	s.logger.Info("Handling TagAvailabilityCheckRequested event", "correlation_id", correlationID)

	// Check tag availability.
	isAvailable, err := s.LeaderboardDB.CheckTagAvailability(ctx, eventPayload.TagNumber)
	if err != nil {
		s.logger.Error("Failed to check tag availability", "error", err, "correlation_id", correlationID)
		return fmt.Errorf("failed to check tag availability: %w", err)
	}

	// If tag is available, publish TagAssignmentRequested.
	if isAvailable {
		assignmentID := uuid.NewString() // Generate a unique ID for the assignment
		s.logger.Info("Tag is available, publishing TagAssignmentRequested event",
			"correlation_id", correlationID,
			"tag_number", eventPayload.TagNumber,
			"assignment_id", assignmentID,
		)

		if err := s.publishTagAssignmentRequested(ctx, msg, eventPayload.DiscordID, eventPayload.TagNumber, assignmentID); err != nil {
			return fmt.Errorf("failed to publish TagAssignmentRequested event: %w", err)
		}
	} else {
		// If tag is not available, publish TagUnavailable event.
		s.logger.Info("Tag is not available",
			"correlation_id", correlationID,
			"tag_number", eventPayload.TagNumber,
		)

		if err := s.publishTagUnavailable(ctx, msg, eventPayload.TagNumber, eventPayload.DiscordID, "Tag is already assigned"); err != nil {
			return fmt.Errorf("failed to publish TagUnavailable event: %w", err)
		}
	}

	return nil
}

// publishTagAvailable publishes a TagAvailable event to the User module.
func (s *LeaderboardService) PublishTagAvailable(ctx context.Context, msg *message.Message, payload *leaderboardevents.TagAssignedPayload) error {
	// Construct the payload for the user.tag.available event
	userEventPayload := &leaderboardevents.TagAvailablePayload{
		DiscordID:    leaderboardtypes.DiscordID(payload.DiscordID),
		TagNumber:    payload.TagNumber,
		AssignmentID: payload.AssignmentID,
	}

	// Publish the new message to the user.tag.available topic
	if err := s.publishEvent(msg, leaderboardevents.TagAvailable, userEventPayload); err != nil {
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

// publishTagAssignmentRequested publishes a TagAssignmentRequested event.
func (s *LeaderboardService) publishTagAssignmentRequested(ctx context.Context, msg *message.Message, discordID leaderboardtypes.DiscordID, tagNumber int, assignmentID string) error {
	eventPayload := &leaderboardevents.TagAssignmentRequestedPayload{
		DiscordID:  discordID,
		TagNumber:  tagNumber,
		UpdateID:   assignmentID,
		Source:     "user", // Assuming this event is triggered by user signup
		UpdateType: "new_tag",
	}

	// Publish the event to the LeaderboardStreamName.
	return s.publishEvent(msg, leaderboardevents.LeaderboardTagAssignmentRequested, eventPayload)
}

// publishTagAssigned publishes a TagAssigned event.
func (s *LeaderboardService) publishTagAssigned(ctx context.Context, msg *message.Message, tagNumber int, discordID leaderboardtypes.DiscordID, assignmentID string) error {
	eventPayload := &leaderboardevents.TagAssignedPayload{
		DiscordID:    discordID,
		TagNumber:    tagNumber,
		AssignmentID: assignmentID,
	}

	// Publish the event to the LeaderboardStreamName.
	return s.publishEvent(msg, leaderboardevents.TagAssigned, eventPayload)
}

// publishTagUnavailable publishes a TagUnavailable event to the User module.
func (s *LeaderboardService) publishTagUnavailable(ctx context.Context, msg *message.Message, tagNumber int, discordID leaderboardtypes.DiscordID, reason string) error {
	eventPayload := &leaderboardevents.TagUnavailablePayload{
		DiscordID: discordID,
		TagNumber: tagNumber,
		Reason:    reason,
	}

	// Publish the event to the UserStreamName.
	return s.publishEvent(msg, leaderboardevents.TagUnavailable, eventPayload)
}

func (s *LeaderboardService) publishTagAssignmentFailed(_ context.Context, msg *message.Message, discordID string, tagNumber int, updateID string, source string, updateType string, reason string) error {
	eventPayload := &leaderboardevents.TagAssignmentFailedPayload{
		DiscordID:  leaderboardtypes.DiscordID(discordID),
		TagNumber:  tagNumber,
		UpdateID:   updateID,
		Source:     source,
		UpdateType: updateType,
		Reason:     reason,
	}

	return s.publishEvent(msg, leaderboardevents.LeaderboardTagAssignmentFailed, eventPayload)
}
