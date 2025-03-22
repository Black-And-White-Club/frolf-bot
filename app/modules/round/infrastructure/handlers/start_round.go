package roundhandlers

import (
	"fmt"
	"log/slog"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
)

func (h *RoundHandlers) HandleRoundStarted(msg *message.Message) ([]*message.Message, error) {
	correlationID, _, err := eventutil.UnmarshalPayload[roundevents.RoundStartedPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal RoundStartedPayload: %w", err)
	}

	h.logger.Info("Received RoundStarted event",
		slog.String("correlation_id", correlationID),
	)

	if err := h.RoundService.ProcessRoundStart(msg); err != nil {
		h.logger.Error("Failed to handle RoundStarted event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to handle RoundStarted event: %w", err)
	}

	h.logger.Info("RoundStarted event processed", slog.String("correlation_id", correlationID))
	return nil
}
