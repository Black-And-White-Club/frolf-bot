package userhandlers

import (
	"fmt"
	"log/slog"

	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	"github.com/Black-And-White-Club/tcr-bot/internal/eventutil"
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

	// Your existing message handling code
	correlationID, payload, err := eventutil.UnmarshalPayload[userevents.UserSignupRequestPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal UserSignupRequest event: %w", err)
	}

	h.logger.Info("Received UserSignupRequest event",
		slog.String("correlation_id", correlationID),
		slog.String("discord_id", string(payload.DiscordID)),
	)

	// Call CreateUser with the extracted information
	if err := h.userService.CreateUser(msg.Context(), msg, payload.DiscordID, &payload.TagNumber); err != nil {
		h.logger.Error("Error in user signup",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to process UserSignupRequest: %w", err)
	}

	h.logger.Info("User signup request processed CALLED FROM HANDLER", slog.String("correlation_id", correlationID))

	return nil
}

// HandleUserCreated handles the UserCreated event.
func (h *UserHandlers) HandleUserCreated(msg *message.Message) error {
	correlationID, payload, err := eventutil.UnmarshalPayload[userevents.UserCreatedPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal UserCreated event: %w", err)
	}

	h.logger.Info("Received UserCreated event",
		slog.String("correlation_id", correlationID),
		slog.String("discord_id", string(payload.DiscordID)),
	)

	// Now, instead of this handler sending a Discord message, we assume that
	// the Discord bot itself is subscribed to the UserCreated event and
	// will handle sending a message to the user.

	return nil
}

// HandleUserCreationFailed handles the UserCreationFailed event.
func (h *UserHandlers) HandleUserCreationFailed(msg *message.Message) error {
	h.logger.Info("HandleUserCreationFailed called", "message_id", msg.UUID)
	correlationID, payload, err := eventutil.UnmarshalPayload[userevents.UserCreationFailedPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal UserCreationFailed event: %w", err)
	}

	h.logger.Error("Received UserCreationFailed event",
		slog.String("correlation_id", correlationID),
		slog.String("reason", payload.Reason),
	)

	// Here, you could potentially store the error information temporarily
	// if needed for future reference or retries. Since the Discord bot
	// listens to UserCreationFailed, it will handle notifying the user.

	return nil
}
