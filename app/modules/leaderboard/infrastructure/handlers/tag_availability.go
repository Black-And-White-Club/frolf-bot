package leaderboardhandlers

import (
	"fmt"
	"log/slog"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleTagAvailabilityCheckRequested handles the TagAvailabilityCheckRequested event.
func (h *LeaderboardHandlers) HandleTagAvailabilityCheckRequested(msg *message.Message) error {
	correlationID, payload, err := eventutil.UnmarshalPayload[leaderboardevents.TagAvailabilityCheckRequestedPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal TagAvailabilityCheckRequestedPayload: %w", err)
	}

	h.logger.Info("Received TagAvailabilityCheckRequested event",
		slog.String("correlation_id", correlationID),
		slog.Int("tag_number", payload.TagNumber),
		slog.String("discord_id", string(payload.DiscordID)),
	)

	// Call the service function to handle the event
	if err := h.leaderboardService.TagAvailabilityCheckRequested(msg.Context(), msg); err != nil {
		h.logger.Error("Failed to handle TagAvailabilityCheckRequested event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to handle TagAvailabilityCheckRequested event: %w", err)
	}

	h.logger.Info("TagAvailabilityCheckRequested event processed", slog.String("correlation_id", correlationID))
	return nil
}
