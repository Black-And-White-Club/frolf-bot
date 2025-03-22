package leaderboardhandlers

import (
	"fmt"
	"log/slog"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleRoundFinalized handles the RoundFinalized event.
func (h *LeaderboardHandlers) HandleRoundFinalized(msg *message.Message) error {
	correlationID, payload, err := eventutil.UnmarshalPayload[leaderboardevents.RoundFinalizedPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal RoundFinalizedPayload: %w", err)
	}

	h.logger.Info("Received RoundFinalized event",
		slog.String("correlation_id", correlationID),
		slog.Any("round_id", roundtypes.ID(payload.RoundID)),
	)

	// Call the service function directly
	if err := h.leaderboardService.RoundFinalized(msg.Context(), msg); err != nil {
		h.logger.Error("Failed to handle RoundFinalized event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to handle RoundFinalized event: %w", err)
	}

	h.logger.Info("RoundFinalized event processed", slog.String("correlation_id", correlationID))
	return nil
}

// HandleLeaderboardUpdateRequested handles the LeaderboardUpdateRequested event.
func (h *LeaderboardHandlers) HandleLeaderboardUpdateRequested(msg *message.Message) error {
	correlationID, payload, err := eventutil.UnmarshalPayload[leaderboardevents.LeaderboardUpdateRequestedPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal LeaderboardUpdateRequestedPayload: %w", err)
	}

	h.logger.Info("Received LeaderboardUpdateRequested event",
		slog.String("correlation_id", correlationID),
		slog.Any("round_id", payload.RoundID),
	)

	// Extract the context from the message
	ctx := msg.Context()

	// Call the service function to handle the event
	if err := h.leaderboardService.LeaderboardUpdateRequested(ctx, msg); err != nil {
		h.logger.Error("Failed to handle LeaderboardUpdateRequested event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to handle LeaderboardUpdateRequested event: %w", err)
	}

	h.logger.Info("LeaderboardUpdateRequested event processed", slog.String("correlation_id", correlationID))
	return nil
}
