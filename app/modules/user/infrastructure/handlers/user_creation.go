package userhandlers

import (
	"fmt"
	"log/slog"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

// HandleUserSignupRequest handles the UserSignupRequest event.
func (h *UserHandlers) HandleUserSignupRequest(msg *message.Message) error {
	h.logger.Info("HandleUserSignupRequest triggered",
		slog.String("message_id", msg.UUID),
		slog.String("correlation_id", msg.Metadata.Get(middleware.CorrelationIDMetadataKey)),
	)

	// Log the entire message for debugging
	h.logger.Debug("Message details", slog.Any("message", msg))

	correlationID, payload, err := eventutil.UnmarshalPayload[userevents.UserSignupRequestPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal UserSignupRequest event: %w", err)
	}

	h.logger.Info("Received UserSignupRequest event",
		slog.String("correlation_id", correlationID),
		slog.String("user_id", string(payload.DiscordID)),
	)

	// If a tag is provided, check its availability
	if payload.TagNumber != nil {
		// Call CheckTagAvailability with the extracted information
		if err := h.userService.CheckTagAvailability(msg.Context(), msg, *payload.TagNumber, payload.DiscordID); err != nil {
			h.logger.Error("Failed to check tag availability",
				slog.String("correlation_id", correlationID),
				slog.Any("error", err),
			)
			return fmt.Errorf("failed to check tag availability: %w", err)
		}

		h.logger.Info("Tag availability check requested", slog.String("correlation_id", correlationID))
		return nil // Exit here, waiting for the response
	}

	// If no tag is provided, proceed with user creation
	if err := h.userService.CreateUser(msg.Context(), msg, payload.DiscordID, nil); err != nil {
		h.logger.Error("Error in user signup",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to process UserSignupRequest: %w", err)
	}

	h.logger.Info("User signup request processed", slog.String("correlation_id", correlationID))

	return nil
}
