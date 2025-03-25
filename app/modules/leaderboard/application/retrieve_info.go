package leaderboardservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleGetLeaderboardRequest handles the GetLeaderboardRequest event.
func (s *LeaderboardService) GetLeaderboardRequest(ctx context.Context, msg *message.Message) error {
	correlationID, _, err := eventutil.UnmarshalPayload[leaderboardevents.GetLeaderboardRequestPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal GetLeaderboardRequestPayload: %w", err)
	}

	s.logger.Info("Handling GetLeaderboardRequest event", "correlation_id", correlationID)

	// 1. Get the active leaderboard from the database.
	leaderboard, err := s.LeaderboardDB.GetActiveLeaderboard(ctx)
	if err != nil {
		s.logger.Error("Failed to get active leaderboard", "error", err, "correlation_id", correlationID)
		return fmt.Errorf("failed to get active leaderboard: %w", err)
	}

	// 2. Prepare the response payload.
	leaderboardEntries := make([]leaderboardevents.LeaderboardEntry, 0, len(leaderboard.LeaderboardData))
	for tagNumber, UserID := range leaderboard.LeaderboardData {
		leaderboardEntries = append(leaderboardEntries, leaderboardevents.LeaderboardEntry{
			TagNumber: &tagNumber,
			UserID:    leaderboardtypes.UserID(UserID),
		})
	}

	responsePayload := leaderboardevents.GetLeaderboardResponsePayload{
		Leaderboard: leaderboardEntries,
	}

	// 3. Publish the GetLeaderboardResponse event.
	return s.publishGetLeaderboardResponse(ctx, msg, responsePayload)
}

// publishGetLeaderboardResponse publishes a GetLeaderboardResponse event.
func (s *LeaderboardService) publishGetLeaderboardResponse(_ context.Context, msg *message.Message, responsePayload leaderboardevents.GetLeaderboardResponsePayload) error {
	payloadBytes, err := json.Marshal(responsePayload)
	if err != nil {
		return fmt.Errorf("failed to marshal GetLeaderboardResponsePayload: %w", err)
	}

	newMessage := message.NewMessage(watermill.NewUUID(), payloadBytes)
	s.eventUtil.PropagateMetadata(msg, newMessage)

	if err := s.EventBus.Publish(leaderboardevents.GetLeaderboardResponse, newMessage); err != nil {
		return fmt.Errorf("failed to publish GetLeaderboardResponse event: %w", err)
	}

	return nil
}

// GetTagByUserIDRequest handles the GetTagByUserIDRequest event.
func (s *LeaderboardService) GetTagByUserIDRequest(ctx context.Context, msg *message.Message) error {
	correlationID, eventPayload, err := eventutil.UnmarshalPayload[leaderboardevents.TagNumberRequestPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal GetTagByUserIDRequestPayload: %w", err)
	}

	s.logger.Info("✅ Inside GetTagByUserIDRequest",
		slog.String("correlation_id", correlationID),
		slog.String("user_id", string(eventPayload.UserID)))

	// Fetch tag number from DB
	tagNumber, err := s.LeaderboardDB.GetTagByUserID(ctx, string(eventPayload.UserID))
	if err != nil {
		s.logger.Error("❌ Failed to get tag by UserID",
			slog.String("error", err.Error()))
		return fmt.Errorf("failed to get tag by UserID: %w", err)
	}

	// ✅ Properly handle `nil` case
	var tagPtr *int
	if tagNumber != nil {
		tagValue := *tagNumber // Dereference to avoid nil
		tagPtr = &tagValue     // Create a new pointer
	}

	s.logger.Info("📊 Retrieved tag number",
		slog.Any("tag_number", tagPtr)) // Debugging

	// ✅ Publish response with correct tag number handling
	return s.publishGetTagByUserIDResponse(ctx, msg, leaderboardevents.GetTagNumberResponsePayload{
		TagNumber: tagPtr, // Now correctly represents "has no tag"
		RoundID:   eventPayload.RoundID,
		UserID:    eventPayload.UserID,
	})
}

// publishGetTagByUserIDResponse publishes a GetTagByUserIDResponse event.
func (s *LeaderboardService) publishGetTagByUserIDResponse(ctx context.Context, msg *message.Message, responsePayload leaderboardevents.GetTagNumberResponsePayload) error {
	// 🔥 Make sure we log nil safely
	var tagStr string
	if responsePayload.TagNumber == nil {
		tagStr = "nil"
	} else {
		tagStr = fmt.Sprintf("%d", *responsePayload.TagNumber)
	}

	s.logger.Info("📤 Publishing GetTagByUserIDResponse",
		slog.String("tag_number", tagStr)) // Safe nil logging

	payloadBytes, err := json.Marshal(responsePayload)
	if err != nil {
		return fmt.Errorf("failed to marshal GetTagByUserIDResponsePayload: %w", err)
	}

	newMessage := message.NewMessage(watermill.NewUUID(), payloadBytes)

	if err := s.EventBus.Publish(leaderboardevents.GetTagByUserIDResponse, newMessage); err != nil {
		s.logger.Error("❌ Failed to publish GetTagByUserIDResponse",
			slog.Any("error", err))
		return fmt.Errorf("failed to publish GetTagByUserIDResponse event: %w", err)
	}

	s.logger.Info("✅ Successfully published GetTagByUserIDResponse")
	return nil
}
