package roundhandlers

import (
	"fmt"
	"log/slog"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
)

func (h *RoundHandlers) HandleRoundReminder(msg *message.Message) ([]*message.Message, error) {
	correlationID, _, err := eventutil.UnmarshalPayload[roundevents.DiscordReminderPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal RoundReminderPayload: %w", err)
	}

	h.logger.Info("Received RoundReminder event",
		slog.String("correlation_id", correlationID),
	)

	if err := h.RoundService.ProcessRoundReminder(msg.Context(), msg); err != nil {
		h.logger.Error("Failed to handle RoundReminder event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to handle RoundReminder event: %w", err)
	}

	h.logger.Info("RoundReminder event processed", slog.String("correlation_id", correlationID))
	return nil
}
