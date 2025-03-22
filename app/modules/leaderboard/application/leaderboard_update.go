package leaderboardservice

import (
	"context"
	"fmt"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	leaderboarddomain "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/domain"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill"
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
func (s *LeaderboardService) publishLeaderboardUpdateRequested(_ context.Context, msg *message.Message, roundID roundtypes.ID, sortedParticipantTags []string) error {
	eventPayload := leaderboardevents.LeaderboardUpdateRequestedPayload{
		RoundID:               roundID,
		SortedParticipantTags: sortedParticipantTags,
		Source:                "round", // Source is "round" for round-based updates
		UpdateID:              watermill.NewUUID(),
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

	// 1. Get the current active leaderboard.
	currentLeaderboard, err := s.LeaderboardDB.GetActiveLeaderboard(ctx)
	if err != nil {
		s.logger.Error("Failed to get active leaderboard", "error", err, "correlation_id", correlationID)
		return fmt.Errorf("failed to get active leaderboard: %w", err) // Handle this error according to your application's needs
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
	newLeaderboard := &leaderboarddb.Leaderboard{
		LeaderboardData:   updatedLeaderboard,
		IsActive:          true,
		ScoreUpdateSource: source,
		ScoreUpdateID:     eventPayload.UpdateID,
	}

	newLeaderboardID, err := s.LeaderboardDB.CreateLeaderboard(ctx, newLeaderboard)
	if err != nil {
		s.logger.Error("Failed to create new leaderboard", "error", err, "correlation_id", correlationID)
		// Publish LeaderboardUpdateFailed event
		if pubErr := s.publishLeaderboardUpdateFailed(ctx, msg, roundtypes.ID(eventPayload.RoundID), err.Error()); pubErr != nil {
			s.logger.Error("Failed to publish LeaderboardUpdateFailed event", "error", pubErr, "correlation_id", correlationID)
		}
		return nil // Error handled gracefully
	}

	// Store the ID inside the leaderboard object
	newLeaderboard.ID = newLeaderboardID

	// 5. Deactivate the current leaderboard entry.
	if err := s.LeaderboardDB.DeactivateLeaderboard(ctx, currentLeaderboard.ID); err != nil {
		s.logger.Error("Failed to deactivate current leaderboard", "error", err, "correlation_id", correlationID)
		// Publish LeaderboardUpdateFailed event
		if pubErr := s.publishLeaderboardUpdateFailed(ctx, msg, eventPayload.RoundID, err.Error()); pubErr != nil {
			s.logger.Error("Failed to publish LeaderboardUpdateFailed event", "error", pubErr, "correlation_id", correlationID)
		}
		return nil // Error handled gracefully
	}

	// 6. Publish LeaderboardUpdated with full leaderboard data
	if err := s.publishLeaderboardUpdated(ctx, msg, newLeaderboard, eventPayload.RoundID); err != nil {
		s.logger.Error("Failed to publish LeaderboardUpdated event", "error", err, "correlation_id", correlationID)
		// Publish LeaderboardUpdateFailed event
		if pubErr := s.publishLeaderboardUpdateFailed(ctx, msg, eventPayload.RoundID, err.Error()); pubErr != nil {
			s.logger.Error("Failed to publish LeaderboardUpdateFailed event", "error", pubErr, "correlation_id", correlationID)
		}
		return nil // Error handled gracefully
	}

	return nil
}

// publishLeaderboardUpdated publishes a LeaderboardUpdated event.
func (s *LeaderboardService) publishLeaderboardUpdated(_ context.Context, msg *message.Message, leaderboard *leaderboarddb.Leaderboard, roundID roundtypes.ID) error {
	eventPayload := leaderboardevents.LeaderboardUpdatedPayload{
		LeaderboardID:   leaderboard.ID,
		RoundID:         roundID,
		LeaderboardData: leaderboard.LeaderboardData, // Pass full leaderboard data
	}

	return s.publishEvent(msg, leaderboardevents.LeaderboardUpdated, eventPayload)
}

// publishLeaderboardUpdateFailed publishes a LeaderboardUpdateFailed event.
func (s *LeaderboardService) publishLeaderboardUpdateFailed(_ context.Context, msg *message.Message, roundID roundtypes.ID, reason string) error {
	eventPayload := leaderboardevents.LeaderboardUpdateFailedPayload{
		RoundID: roundID,
		Reason:  reason,
	}

	return s.publishEvent(msg, leaderboardevents.LeaderboardUpdateFailed, eventPayload)
}
