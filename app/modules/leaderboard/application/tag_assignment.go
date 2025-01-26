package leaderboardservice

import (
	"context"
	"fmt"

	leaderboardevents "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/domain/events"
	leaderboarddb "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/Black-And-White-Club/tcr-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
)

// -- Tag Assignment --

// HandleTagAssigned handles the TagAssigned event from the User module.
func (s *LeaderboardService) TagAssigned(ctx context.Context, msg *message.Message) error {
	correlationID, eventPayload, err := eventutil.UnmarshalPayload[leaderboardevents.TagAssignedPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal TagAssignedPayload: %w", err)
	}

	s.logger.Info("Handling TagAssigned event", "correlation_id", correlationID)

	// Check if the tag is available.
	isAvailable, err := s.LeaderboardDB.CheckTagAvailability(ctx, eventPayload.TagNumber)
	if err != nil {
		s.logger.Error("Failed to check tag availability", "error", err, "correlation_id", correlationID)
		return fmt.Errorf("failed to check tag availability: %w", err)
	}
	if !isAvailable {
		s.logger.Error("Tag is not available", "tag_number", eventPayload.TagNumber, "correlation_id", correlationID)
		return fmt.Errorf("tag number %d is already assigned", eventPayload.TagNumber)
	}

	// Publish TagAssignmentRequested event
	return s.publishTagAssignmentRequested(ctx, msg, eventPayload.DiscordID, eventPayload.TagNumber, correlationID)
}

// TagAssignmentRequested handles the TagAssignmentRequested event.
func (s *LeaderboardService) TagAssignmentRequested(ctx context.Context, msg *message.Message) error {
	correlationID, eventPayload, err := eventutil.UnmarshalPayload[leaderboardevents.TagAssignmentRequestedPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal TagAssignmentRequestedPayload: %w", err)
	}

	s.logger.Info("Handling TagAssignmentRequested event", "correlation_id", correlationID)

	// Assign the tag to the user in the leaderboard.
	err = s.LeaderboardDB.AssignTag(ctx, eventPayload.DiscordID, eventPayload.TagNumber, leaderboarddb.ServiceUpdateTagSourceCreateUser, eventPayload.UpdateID)
	if err != nil {
		s.logger.Error("Failed to assign tag to user", "error", err, "correlation_id", correlationID)
		// Publish a TagAssignmentFailed event here.
		if pubErr := s.publishTagAssignmentFailed(ctx, msg, string(eventPayload.DiscordID), eventPayload.TagNumber, eventPayload.UpdateID, "user", "new_tag", err.Error()); pubErr != nil {
			return fmt.Errorf("failed to publish TagAssignmentFailed event: %w", err)
		}
		return fmt.Errorf("failed to assign tag to user: %w", err)
	}

	// Publish TagAssigned event to notify that the tag has been assigned.
	if err := s.publishTagAssigned(ctx, msg, eventPayload.TagNumber, eventPayload.DiscordID, eventPayload.UpdateID); err != nil {
		return fmt.Errorf("failed to publish TagAssigned event: %w", err)
	}

	s.logger.Info("Tag assigned and leaderboard updated successfully", "correlation_id", correlationID)
	return nil
}
