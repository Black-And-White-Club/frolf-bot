package leaderboardhandlers

import (
	"fmt"
	"log/slog"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleGetLeaderboardRequest handles the GetLeaderboardRequest event.
func (h *LeaderboardHandlers) HandleGetLeaderboardRequest(msg *message.Message) ([]*message.Message, error) {
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

// HandleGetTagByUserIDRequest handles the GetTagByUserIDRequest event.
func (h *LeaderboardHandlers) HandleGetTagByUserIDRequest(msg *message.Message) ([]*message.Message, error) {
	correlationID, payload, err := eventutil.UnmarshalPayload[leaderboardevents.TagNumberRequestPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal GetTagByUserIDRequestPayload: %w", err)
	}

	h.logger.Info("✅ Inside HandleGetTagByUserIDRequest",
		slog.String("correlation_id", correlationID),
		slog.String("user_id", string(payload.UserID)))

	if err := h.leaderboardService.GetTagByUserIDRequest(msg.Context(), msg); err != nil {
		h.logger.Error("❌ Failed to handle GetTagByUserIDRequest event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err))

		// ❌ **DON'T ACKNOWLEDGE on failure!** Let NATS retry.
		return err
	}

	// ✅ **Manually acknowledge the message**
	msg.Ack()

	return nil
}
