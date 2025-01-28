package leaderboardservice

import (
	"context"
	"fmt"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	leaderboarddomain "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/domain"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/domain/types"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
)

// -- Tag Swap --

// HandleTagSwapRequested handles the TagSwapRequested event.
func (s *LeaderboardService) TagSwapRequested(ctx context.Context, msg *message.Message) error {
	correlationID, eventPayload, err := eventutil.UnmarshalPayload[leaderboardevents.TagSwapRequestedPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal TagSwapRequestedPayload: %w", err)
	}

	s.logger.Info("Handling TagSwapRequested event", "correlation_id", correlationID)

	// 1. Get the current leaderboard.
	currentLeaderboard, err := s.LeaderboardDB.GetActiveLeaderboard(ctx)
	if err != nil {
		s.logger.Error("Failed to get active leaderboard", "error", err, "correlation_id", correlationID)
		return fmt.Errorf("failed to get active leaderboard: %w", err)
	}

	// 2. Check if both requestorID and targetID have tags on the leaderboard.
	_, requestorExists := leaderboarddomain.FindTagByDiscordID(currentLeaderboard, leaderboardtypes.DiscordID(eventPayload.RequestorID))
	_, targetExists := leaderboarddomain.FindTagByDiscordID(currentLeaderboard, leaderboardtypes.DiscordID(eventPayload.TargetID))

	if !requestorExists || !targetExists {
		s.logger.Error("One or both users do not have tags on the leaderboard", "requestor", eventPayload.RequestorID, "target", eventPayload.TargetID, "correlation_id", correlationID)
		return fmt.Errorf("one or both users do not have tags on the leaderboard")
	}

	// Publish TagSwapInitiated event
	return s.publishTagSwapInitiated(ctx, msg, eventPayload.RequestorID, eventPayload.TargetID)
}

// publishTagSwapInitiated publishes a TagSwapInitiated event.
func (s *LeaderboardService) publishTagSwapInitiated(_ context.Context, msg *message.Message, requestorID, targetID string) error {
	eventPayload := leaderboardevents.TagSwapInitiatedPayload{
		RequestorID: string(requestorID),
		TargetID:    targetID,
	}

	return s.publishEvent(msg, leaderboardevents.TagSwapInitiated, eventPayload)
}

// HandleTagSwapInitiated handles the TagSwapInitiated event.
func (s *LeaderboardService) TagSwapInitiated(ctx context.Context, msg *message.Message) error {
	correlationID, eventPayload, err := eventutil.UnmarshalPayload[leaderboardevents.TagSwapInitiatedPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal TagSwapInitiatedPayload: %w", err)
	}

	s.logger.Info("Handling TagSwapInitiated event", "correlation_id", correlationID)

	// Perform the tag swap in the database.
	if err := s.LeaderboardDB.SwapTags(ctx, eventPayload.RequestorID, eventPayload.TargetID); err != nil {
		s.logger.Error("Failed to swap tags in DB", "error", err, "correlation_id", correlationID)

		// Publish TagSwapFailed event
		if pubErr := s.publishTagSwapFailed(ctx, msg, leaderboardtypes.DiscordID(eventPayload.RequestorID), eventPayload.TargetID, err.Error()); pubErr != nil {
			s.logger.Error("Failed to publish TagSwapFailed event", "error", pubErr, "correlation_id", correlationID)
		}

		return fmt.Errorf("failed to swap tags in DB: %w", err) // Now we return the error
	}

	// Publish TagSwapProcessed event
	if err := s.publishTagSwapProcessed(ctx, msg, leaderboardtypes.DiscordID(eventPayload.RequestorID), eventPayload.TargetID); err != nil {
		s.logger.Error("Failed to publish TagSwapProcessed event", "error", err, "correlation_id", correlationID)
	}

	s.logger.Info("Tags swapped successfully", "correlation_id", correlationID)
	return nil
}

// publishTagSwapProcessed publishes a TagSwapProcessed event.
func (s *LeaderboardService) publishTagSwapProcessed(_ context.Context, msg *message.Message, requestorID leaderboardtypes.DiscordID, targetID string) error {
	eventPayload := leaderboardevents.TagSwapProcessedPayload{
		RequestorID: string(requestorID),
		TargetID:    targetID,
	}

	return s.publishEvent(msg, leaderboardevents.TagSwapProcessed, eventPayload)
}

// publishTagSwapFailed publishes a TagSwapFailed event.
func (s *LeaderboardService) publishTagSwapFailed(_ context.Context, msg *message.Message, requestorID leaderboardtypes.DiscordID, targetID string, reason string) error {
	eventPayload := leaderboardevents.TagSwapFailedPayload{
		RequestorID: string(requestorID),
		TargetID:    targetID,
		Reason:      reason,
	}

	return s.publishEvent(msg, leaderboardevents.TagSwapFailed, eventPayload)
}
