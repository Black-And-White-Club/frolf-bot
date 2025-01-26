package leaderboardservice

import (
	"context"
	"fmt"

	leaderboarddomain "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/domain"
	leaderboardevents "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/domain/events"
	leaderboarddb "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/Black-And-White-Club/tcr-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
)

// -- Leaderboard Update --

// HandleRoundFinalized handles the RoundFinalized event from the Score module.
func (s *LeaderboardService) RoundFinalized(ctx context.Context, msg *message.Message) error {
	correlationID, eventPayload, err := eventutil.UnmarshalPayload[leaderboardevents.RoundFinalizedPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal RoundFinalizedPayload: %w", err)
	}

	s.logger.Info("Handling RoundFinalized event", "correlation_id", correlationID)

	// Publish a LeaderboardUpdateRequested event (internal to the Leaderboard module)
	return s.publishLeaderboardUpdateRequested(ctx, msg, eventPayload.RoundID, eventPayload.SortedParticipantTags)
}

// publishLeaderboardUpdateRequested publishes a LeaderboardUpdateRequested event.
func (s *LeaderboardService) publishLeaderboardUpdateRequested(_ context.Context, msg *message.Message, roundID string, sortedParticipantTags []string) error {
	eventPayload := leaderboardevents.LeaderboardUpdateRequestedPayload{
		RoundID:               roundID,
		SortedParticipantTags: sortedParticipantTags,
		Source:                "round", // Source is "round" for round-based updates
		UpdateID:              roundID, // Update ID is the round ID
	}

	return s.publishEvent(msg, leaderboardevents.LeaderboardUpdateRequested, eventPayload)
}

// HandleLeaderboardUpdateRequested handles the LeaderboardUpdateRequested event.
func (s *LeaderboardService) LeaderboardUpdateRequested(ctx context.Context, msg *message.Message) error {
	correlationID, eventPayload, err := eventutil.UnmarshalPayload[leaderboardevents.LeaderboardUpdateRequestedPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal LeaderboardUpdateRequestedPayload: %w", err)
	}

	s.logger.Info("Handling LeaderboardUpdateRequested event", "correlation_id", correlationID)

	// 1. Get the current leaderboard.
	currentLeaderboard, err := s.LeaderboardDB.GetActiveLeaderboard(ctx)
	if err != nil {
		s.logger.Error("Failed to get active leaderboard", "error", err, "correlation_id", correlationID)
		return fmt.Errorf("failed to get active leaderboard: %w", err)
	}

	// 2. Generate the updated leaderboard.
	updatedLeaderboard := leaderboarddomain.GenerateUpdatedLeaderboard(currentLeaderboard, eventPayload.SortedParticipantTags)

	// 3. Determine the source and update ID for the database update
	var source leaderboarddb.ServiceUpdateTagSource
	switch eventPayload.Source {
	case "round":
		source = leaderboarddb.ServiceUpdateTagSourceProcessScores
	case "manual":
		source = leaderboarddb.ServiceUpdateTagSourceManual
	default:
		s.logger.Error("Invalid source for leaderboard update", "source", eventPayload.Source, "correlation_id", correlationID)
		return fmt.Errorf("invalid source for leaderboard update: %s", eventPayload.Source)
	}

	// 4. Create new leaderboard in the database.
	newLeaderboardID, err := s.LeaderboardDB.CreateLeaderboard(ctx, &leaderboarddb.Leaderboard{
		LeaderboardData:   updatedLeaderboard,
		IsActive:          true,
		ScoreUpdateSource: source,
		ScoreUpdateID:     eventPayload.UpdateID,
	})
	if err != nil {
		s.logger.Error("Failed to create new leaderboard", "error", err, "correlation_id", correlationID)
		return s.publishLeaderboardUpdateFailed(ctx, msg, eventPayload.RoundID, err.Error())
	}

	// 5. Deactivate the current leaderboard entry.
	if err := s.LeaderboardDB.DeactivateLeaderboard(ctx, currentLeaderboard.ID); err != nil {
		s.logger.Error("Failed to deactivate current leaderboard", "error", err, "correlation_id", correlationID)
		return fmt.Errorf("failed to deactivate current leaderboard: %w", err)
	}

	// 6. Publish LeaderboardUpdated
	if err := s.publishLeaderboardUpdated(ctx, msg, newLeaderboardID, eventPayload.RoundID); err != nil {
		s.logger.Error("Failed to publish LeaderboardUpdated event", "error", err, "correlation_id", correlationID)
	}

	return nil
}

// publishLeaderboardUpdated publishes a LeaderboardUpdated event.
func (s *LeaderboardService) publishLeaderboardUpdated(_ context.Context, msg *message.Message, leaderboardID int64, roundID string) error {
	eventPayload := leaderboardevents.LeaderboardUpdatedPayload{
		LeaderboardID: leaderboardID,
		RoundID:       roundID,
	}

	return s.publishEvent(msg, leaderboardevents.LeaderboardUpdated, eventPayload)
}

// publishLeaderboardUpdateFailed publishes a LeaderboardUpdateFailed event.
func (s *LeaderboardService) publishLeaderboardUpdateFailed(_ context.Context, msg *message.Message, roundID string, reason string) error {
	eventPayload := leaderboardevents.LeaderboardUpdateFailedPayload{
		RoundID: roundID,
		Reason:  reason,
	}

	return s.publishEvent(msg, leaderboardevents.LeaderboardUpdateFailed, eventPayload)
}
