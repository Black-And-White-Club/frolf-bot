package leaderboardservice

import (
	"context"
	"encoding/json"
	"fmt"

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
	for tagNumber, discordID := range leaderboard.LeaderboardData {
		leaderboardEntries = append(leaderboardEntries, leaderboardevents.LeaderboardEntry{
			TagNumber: tagNumber,
			DiscordID: leaderboardtypes.DiscordID(discordID),
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

// HandleGetTagByDiscordIDRequest handles the GetTagByDiscordIDRequest event.
func (s *LeaderboardService) GetTagByDiscordIDRequest(ctx context.Context, msg *message.Message) error {
	correlationID, eventPayload, err := eventutil.UnmarshalPayload[leaderboardevents.GetTagByDiscordIDRequestPayload](msg, s.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal GetTagByDiscordIDRequestPayload: %w", err)
	}

	s.logger.Info("Handling GetTagByDiscordIDRequest event", "correlation_id", correlationID)

	// 1. Get the tag number from the database.
	tagNumber, err := s.LeaderboardDB.GetTagByDiscordID(ctx, string(eventPayload.DiscordID))
	if err != nil {
		s.logger.Error("Failed to get tag by Discord ID", "error", err, "correlation_id", correlationID)
		return fmt.Errorf("failed to get tag by Discord ID: %w", err)
	}

	// 2. Prepare the response payload.
	responsePayload := leaderboardevents.GetTagByDiscordIDResponsePayload{
		TagNumber: tagNumber,
	}

	// 3. Publish the GetTagByDiscordIDResponse event.
	return s.publishGetTagByDiscordIDResponse(ctx, msg, responsePayload)
}

// publishGetTagByDiscordIDResponse publishes a GetTagByDiscordIDResponse event.
func (s *LeaderboardService) publishGetTagByDiscordIDResponse(_ context.Context, msg *message.Message, responsePayload leaderboardevents.GetTagByDiscordIDResponsePayload) error {
	payloadBytes, err := json.Marshal(responsePayload)
	if err != nil {
		return fmt.Errorf("failed to marshal GetTagByDiscordIDResponsePayload: %w", err)
	}

	newMessage := message.NewMessage(watermill.NewUUID(), payloadBytes)
	s.eventUtil.PropagateMetadata(msg, newMessage)

	if err := s.EventBus.Publish(leaderboardevents.GetTagByDiscordIDResponse, newMessage); err != nil {
		return fmt.Errorf("failed to publish GetTagByDiscordIDResponse event: %w", err)
	}

	return nil
}
