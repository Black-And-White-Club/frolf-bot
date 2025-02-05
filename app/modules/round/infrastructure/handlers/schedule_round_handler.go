package roundhandlers

import (
	"fmt"
	"log/slog"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
)

func (h *RoundHandlers) HandleScheduleRoundEvents(msg *message.Message) error {
	correlationID, _, err := eventutil.UnmarshalPayload[roundevents.RoundStoredPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal RoundStoredPayload: %w", err)
	}

	h.logger.Info("Received RoundStored event",
		slog.String("correlation_id", correlationID),
	)

	if err := h.RoundService.ScheduleRoundEvents(msg.Context(), msg); err != nil {
		h.logger.Error("Failed to handle RoundStored event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to handle RoundStored event: %w", err)
	}

	h.logger.Info("RoundStored event processed", slog.String("correlation_id", correlationID))
	return nil
}
