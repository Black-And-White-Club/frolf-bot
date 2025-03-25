package leaderboardhandlers

import (
	"fmt"
	"log/slog"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleTagSwapRequested handles the TagSwapRequested event.
func (h *LeaderboardHandlers) HandleTagSwapRequested(msg *message.Message) ([]*message.Message, error) {
	correlationID, payload, err := eventutil.UnmarshalPayload[leaderboardevents.TagSwapRequestedPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal TagSwapRequestedPayload: %w", err)
	}

	h.logger.Info("Received TagSwapRequested event",
		slog.String("correlation_id", correlationID),
		slog.String("requestor_id", payload.RequestorID),
		slog.String("target_id", payload.TargetID),
	)

	// Call the service function to handle the event
	if err := h.leaderboardService.TagSwapRequested(msg.Context(), msg); err != nil {
		h.logger.Error("Failed to handle TagSwapRequested event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to handle TagSwapRequested event: %w", err)
	}

	h.logger.Info("TagSwapRequested event processed", slog.String("correlation_id", correlationID))
	return nil
}

// HandleTagSwapInitiated handles the TagSwapInitiated event.
func (h *LeaderboardHandlers) HandleTagSwapInitiated(msg *message.Message) ([]*message.Message, error) {
	correlationID, payload, err := eventutil.UnmarshalPayload[leaderboardevents.TagSwapInitiatedPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal TagSwapInitiatedPayload: %w", err)
	}

	h.logger.Info("Received TagSwapInitiated event",
		slog.String("correlation_id", correlationID),
		slog.String("requestor_id", payload.RequestorID),
		slog.String("target_id", payload.TargetID),
	)

	// Call the service function to handle the event
	if err := h.leaderboardService.TagSwapInitiated(msg.Context(), msg); err != nil {
		h.logger.Error("Failed to handle TagSwapInitiated event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to handle TagSwapInitiated event: %w", err)
	}

	h.logger.Info("TagSwapInitiated event processed", slog.String("correlation_id", correlationID))
	return nil
}
