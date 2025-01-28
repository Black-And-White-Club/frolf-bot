package roundhandlers

import (
	"fmt"
	"log/slog"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
)

func (h *RoundHandlers) HandleRoundTagNumberRequest(msg *message.Message) error {
	correlationID, _, err := eventutil.UnmarshalPayload[roundevents.TagNumberRequestPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal TagNumberRequestPayload: %w", err)
	}

	h.logger.Info("Received TagNumberRequest event",
		slog.String("correlation_id", correlationID),
	)

	if err := h.RoundService.TagNumberRequest(msg.Context(), msg); err != nil {
		h.logger.Error("Failed to handle TagNumberRequest event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to handle TagNumberRequest event: %w", err)
	}

	h.logger.Info("TagNumberRequest event processed", slog.String("correlation_id", correlationID))
	return nil
}

func (h *RoundHandlers) HandleLeaderboardGetTagNumberResponse(msg *message.Message) error {
	correlationID, _, err := eventutil.UnmarshalPayload[roundevents.GetTagNumberResponsePayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal GetTagNumberResponsePayload: %w", err)
	}

	h.logger.Info("Received GetTagNumberResponse event",
		slog.String("correlation_id", correlationID),
	)

	if err := h.RoundService.TagNumberResponse(msg.Context(), msg); err != nil {
		h.logger.Error("Failed to handle GetTagNumberResponse event",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to handle GetTagNumberResponse event: %w", err)
	}

	h.logger.Info("GetTagNumberResponse event processed", slog.String("correlation_id", correlationID))
	return nil
}
