package userhandlers

import (
	"fmt"
	"log/slog"

	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	"github.com/Black-And-White-Club/tcr-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleCheckTagAvailabilityRequest handles the CheckTagAvailabilityRequest event.
func (h *UserHandlers) HandleCheckTagAvailabilityRequest(msg *message.Message) error {
	correlationID, payload, err := eventutil.UnmarshalPayload[userevents.CheckTagAvailabilityRequestPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal CheckTagAvailabilityRequest event: %w", err)
	}

	h.logger.Info("Received CheckTagAvailabilityRequest event",
		slog.String("correlation_id", correlationID),
		slog.Int("tag_number", payload.TagNumber),
	)

	// Call the service function to check the tag availability.
	if err := h.userService.CheckTagAvailability(msg.Context(), msg, payload.TagNumber); err != nil {
		h.logger.Error("Failed to check tag availability",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to check tag availability: %w", err)
	}

	h.logger.Info("CheckTagAvailabilityRequest processed", slog.String("correlation_id", correlationID))

	return nil
}
