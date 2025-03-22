package leaderboardservice

import (
	"context"
	"fmt"
	"log/slog"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
)

// -- Tag Assignment --

// TagAssignmentRequested handles the TagAssignmentRequested event.
func (s *LeaderboardService) TagAssignmentRequested(ctx context.Context, msg *message.Message) error {
	correlationID, eventPayload, err := eventutil.UnmarshalPayload[leaderboardevents.TagAssignmentRequestedPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal TagAssignmentRequestedPayload: %w", err)
	}

	s.logger.Info("Handling TagAssignmentRequested event", "correlation_id", correlationID)

	// Assign the tag to the user in the leaderboard.
	err = s.LeaderboardDB.AssignTag(ctx, eventPayload.UserID, *eventPayload.TagNumber, leaderboarddb.ServiceUpdateTagSourceCreateUser, eventPayload.UpdateID)
	if err != nil {
		s.logger.Error("Failed to assign tag to user", "error", err, "correlation_id", correlationID)
		// Publish a TagAssignmentFailed event here.
		if pubErr := s.publishTagAssignmentFailed(ctx, msg, string(eventPayload.UserID), *eventPayload.TagNumber, eventPayload.UpdateID, "user", "new_tag", err.Error()); pubErr != nil {
			return fmt.Errorf("failed to publish TagAssignmentFailed event: %w", err)
		}
		return fmt.Errorf("failed to assign tag to user: %w", err)
	}

	// Publish TagAssigned event
	if err := s.publishTagAssigned(ctx, msg, *eventPayload.TagNumber, eventPayload.UserID, eventPayload.UpdateID); err != nil {
		return fmt.Errorf("failed to publish TagAssigned event: %w", err)
	}

	s.logger.Info("Tag assigned and leaderboard updated successfully", "correlation_id", correlationID)
	return nil
}

// TagAssigned handles the TagAssigned event
func (s *LeaderboardService) TagAssigned(ctx context.Context, msg *message.Message) error {
	correlationID, eventPayload, err := eventutil.UnmarshalPayload[leaderboardevents.TagAssignedPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal TagAssignedPayload: %w", err)
	}

	s.logger.Info("Processing TagAssigned event",
		"correlation_id", correlationID,
		"user_id", eventPayload.UserID,
		"tag_number", eventPayload.TagNumber,
		"assignment_id", eventPayload.AssignmentID,
	)

	// Notify the user module or other downstream systems.
	notificationPayload := &leaderboardevents.TagAvailablePayload{
		UserID:       eventPayload.UserID,
		TagNumber:    eventPayload.TagNumber,
		AssignmentID: eventPayload.AssignmentID,
	}

	if err := s.publishEvent(msg, leaderboardevents.TagAvailable, notificationPayload); err != nil {
		s.logger.Error("Failed to publish TagAvailable event",
			"error", err,
			"correlation_id", correlationID,
			"user_id", eventPayload.UserID,
			"tag_number", eventPayload.TagNumber,
		)
		return fmt.Errorf("failed to publish TagAvailable event: %w", err)
	}

	s.logger.Info("TagAssigned processing completed successfully",
		"correlation_id", correlationID,
		"user_id", eventPayload.UserID,
		"tag_number", eventPayload.TagNumber,
	)

	return nil
}

func (s *LeaderboardService) publishTagAssignmentFailed(_ context.Context, msg *message.Message, userID string, tagNumber int, updateID string, source string, updateType string, reason string) error {
	eventPayload := &leaderboardevents.TagAssignmentFailedPayload{
		UserID:     leaderboardtypes.UserID(userID),
		TagNumber:  &tagNumber,
		UpdateID:   updateID,
		Source:     source,
		UpdateType: updateType,
		Reason:     reason,
	}

	return s.publishEvent(msg, leaderboardevents.LeaderboardTagAssignmentFailed, eventPayload)
}

func (s *LeaderboardService) publishTagAssignmentRequested(_ context.Context, msg *message.Message, userID leaderboardtypes.UserID, tagNumber int, assignmentID string) error {
	eventPayload := &leaderboardevents.TagAssignmentRequestedPayload{
		UserID:     userID,
		TagNumber:  &tagNumber,
		UpdateID:   assignmentID,
		Source:     "user",
		UpdateType: "new_tag",
	}

	s.logger.Info("Publishing TagAssignmentRequested", slog.Any("payload", eventPayload)) // Log the payload

	return s.publishEvent(msg, leaderboardevents.LeaderboardTagAssignmentRequested, eventPayload)
}

func (s *LeaderboardService) publishTagUnavailable(_ context.Context, msg *message.Message, tagNumber int, userID leaderboardtypes.UserID, reason string) error {
	eventPayload := &leaderboardevents.TagUnavailablePayload{
		UserID:    leaderboardtypes.UserID(userID),
		TagNumber: &tagNumber,
		Reason:    reason,
	}

	s.logger.Info("Publishing TagUnavailable", slog.Any("payload", eventPayload)) // Log the payload

	return s.publishEvent(msg, leaderboardevents.TagUnavailable, eventPayload)
}
