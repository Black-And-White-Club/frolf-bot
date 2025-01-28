package userhandlers

import (
	"fmt"
	"log/slog"

	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	"github.com/Black-And-White-Club/tcr-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleTagAvailable handles the TagAvailable event.
func (h *UserHandlers) HandleTagAvailable(msg *message.Message) error {
	correlationID, payload, err := eventutil.UnmarshalPayload[userevents.TagAvailablePayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal TagAvailablePayload: %w", err)
	}

	h.logger.Info("HandleTagAvailable triggered",
		slog.String("correlation_id", correlationID),
		slog.String("message_id", msg.UUID),
	) // Add this log

	h.logger.Info("Received TagAvailable event",
		slog.String("correlation_id", correlationID),
		slog.String("discord_id", string(payload.DiscordID)),
		slog.Int("tag_number", payload.TagNumber),
	)

	// Call the service function to create the user
	if err := h.userService.CreateUser(msg.Context(), msg, payload.DiscordID, &payload.TagNumber); err != nil {
		h.logger.Error("Failed to create user",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to create user: %w", err)
	}

	h.logger.Info("TagAvailable event processed", slog.String("correlation_id", correlationID))
	return nil
}

// HandleTagUnavailable handles the TagUnavailable event.
func (h *UserHandlers) HandleTagUnavailable(msg *message.Message) error {
	correlationID, payload, err := eventutil.UnmarshalPayload[userevents.TagUnavailablePayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal TagUnavailablePayload: %w", err)
	}

	h.logger.Info("Received TagUnavailable event",
		slog.String("correlation_id", correlationID),
		slog.String("discord_id", string(payload.DiscordID)),
		slog.Int("tag_number", payload.TagNumber),
	)

	// Call the service function to handle the tag unavailability
	if err := h.userService.TagUnavailable(msg.Context(), msg, payload.TagNumber, payload.DiscordID); err != nil {
		h.logger.Error("Failed to handle TagUnavailable",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to handle TagUnavailable: %w", err)
	}

	h.logger.Info("TagUnavailable event processed", slog.String("correlation_id", correlationID))
	return nil
}
