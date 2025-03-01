package userhandlers

import (
	"fmt"
	"log/slog"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
)

// TODO Need to fix this because it's checking the role that is sent, not the one required.
func (h *UserHandlers) HandleUserPermissionsCheckRequest(msg *message.Message) error {
	// Unmarshal the payload and extract correlation ID
	correlationID, payload, err := eventutil.UnmarshalPayload[userevents.UserPermissionsCheckRequestPayload](msg, h.logger)
	if err != nil {
		h.logger.Error("Failed to unmarshal UserPermissionsCheckRequest event",
			slog.String("message_id", msg.UUID),
			slog.String("correlation_id", msg.Metadata.Get("correlation_id")),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to unmarshal UserPermissionsCheckRequest event: %w", err)
	}

	h.logger.Info("Received UserPermissionsCheckRequest event",
		slog.String("correlation_id", correlationID),
		slog.String("user_id", string(payload.DiscordID)),
		slog.String("role", string(payload.Role)),
		slog.String("requester_id", string(payload.RequesterID)),
	)

	// Validate the Discord ID
	if payload.DiscordID == "" {
		h.logger.Error("Invalid Discord ID in UserPermissionsCheckRequest event",
			slog.String("correlation_id", correlationID),
			slog.Any("payload", payload),
		)
		return fmt.Errorf("invalid Discord ID")
	}

	// Convert and validate role
	roleEnum, err := usertypes.ParseUserRoleEnum(string(payload.Role))
	if err != nil {
		h.logger.Error("Invalid role in UserPermissionsCheckRequest event",
			slog.String("role", string(payload.Role)),
			slog.String("correlation_id", correlationID),
		)
		return fmt.Errorf("invalid role: %w", err)
	}

	// Log before calling the service
	h.logger.Info("Checking user permissions in database",
		slog.String("correlation_id", correlationID),
		slog.String("user_id", string(payload.DiscordID)),
		slog.String("role", string(payload.Role)),
		slog.String("requester_id", string(payload.RequesterID)),
	)

	// Call the CheckUserPermissionsInDB function
	err = h.userService.CheckUserPermissionsInDB(
		msg.Context(),       // Context
		msg,                 // Pass the full message
		payload.DiscordID,   // usertypes.DiscordID
		roleEnum,            // usertypes.UserRoleEnum
		payload.RequesterID, // string
	)

	if err != nil {
		// Permission check failed
		h.logger.Warn("User permissions check failed", slog.Any("error", err))

		// Publish UserPermissionsCheckResponse with HasPermission: false
		if err := h.userService.PublishUserPermissionsCheckResponse(
			msg.Context(),
			msg,
			payload.DiscordID,
			payload.Role,
			payload.RequesterID,
			false, // HasPermission: false
			err.Error(),
		); err != nil {
			h.logger.Error("Failed to publish UserPermissionsCheckResponse", slog.Any("error", err))
			return fmt.Errorf("failed to publish UserPermissionsCheckResponse: %w", err)
		}

		return nil // Return nil to acknowledge the message
	}

	h.logger.Info("User permissions check request processed successfully",
		slog.String("correlation_id", correlationID),
		slog.String("user_id", string(payload.DiscordID)),
		slog.String("role", string(payload.Role)),
		slog.String("requester_id", string(payload.RequesterID)),
	)

	return nil
}

// HandleUserPermissionsCheckFailed handles the UserPermissionsCheckFailed event.
func (h *UserHandlers) HandleUserPermissionsCheckFailed(msg *message.Message) error {
	correlationID, payload, err := eventutil.UnmarshalPayload[userevents.UserPermissionsCheckFailedPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal UserPermissionsCheckFailed event: %w", err)
	}

	h.logger.Error("User permissions check failed",
		slog.String("correlation_id", correlationID),
		slog.String("user_id", string(payload.DiscordID)),
		slog.String("requester_id", string(payload.RequesterID)),
		slog.String("reason", payload.Reason),
	)

	// Call the service with the necessary arguments
	err = h.userService.PublishUserPermissionsCheckFailed(
		msg.Context(),       // Context
		msg,                 // Message
		payload.DiscordID,   // Discord ID
		payload.Role,        // Role
		payload.RequesterID, // Requester ID
		payload.Reason,      // Reason for failure
	)
	if err != nil {
		// Handle the error from the service call
		h.logger.Error("Failed to publish UserPermissionsCheckFailed event", slog.Any("error", err))
		return fmt.Errorf("failed to publish UserPermissionsCheckFailed event: %w", err)
	}

	return nil
}
