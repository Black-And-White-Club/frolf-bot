package leaderboardhandlers

import (
	"fmt"
	"log/slog"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleGetLeaderboardRequest handles the GetLeaderboardRequest event.
func (h *LeaderboardHandlers) HandleGetLeaderboardRequest(msg *message.Message) error {
	correlationID, _, err := eventutil.UnmarshalPayload[leaderboardevents.GetLeaderboardRequestPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal GetLeaderboardRequestPayload: %w", err)
	}

	h.logger.Info("Received GetLeaderboardRequest event",
		slog.String("correlation_id", correlationID),
	)

	// Call the service function to handle the event
	if err := h.leaderboardService.GetLeaderboardRequest(msg.Context(), msg); err != nil {
		h.logger.Error("Failed to handle GetLeaderboardRequest event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to handle GetLeaderboardRequest event: %w", err)
	}

	h.logger.Info("GetLeaderboardRequest event processed", slog.String("correlation_id", correlationID))
	return nil
}

// HandleGetTagByDiscordIDRequest handles the GetTagByDiscordIDRequest event.
func (h *LeaderboardHandlers) HandleGetTagByDiscordIDRequest(msg *message.Message) error {
	correlationID, payload, err := eventutil.UnmarshalPayload[leaderboardevents.GetTagByDiscordIDRequestPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal GetTagByDiscordIDRequestPayload: %w", err)
	}

	h.logger.Info("Received GetTagByDiscordIDRequest event",
		slog.String("correlation_id", correlationID),
		slog.String("discord_id", string(payload.DiscordID)),
	)

	// Call the service function to handle the event
	if err := h.leaderboardService.GetTagByDiscordIDRequest(msg.Context(), msg); err != nil {
		h.logger.Error("Failed to handle GetTagByDiscordIDRequest event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to handle GetTagByDiscordIDRequest event: %w", err)
	}

	h.logger.Info("GetTagByDiscordIDRequest event processed", slog.String("correlation_id", correlationID))
	return nil
}
