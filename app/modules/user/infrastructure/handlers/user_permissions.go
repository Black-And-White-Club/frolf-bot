package userhandlers

import (
	"fmt"
	"log/slog"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
)

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
		slog.String("discord_id", string(payload.DiscordID)),
		slog.String("role", string(payload.Role)),
		slog.String("requester_id", payload.RequesterID),
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
		slog.String("discord_id", string(payload.DiscordID)),
		slog.String("role", string(payload.Role)),
		slog.String("requester_id", payload.RequesterID),
	)

	// Call the CheckUserPermissionsInDB function
	err = h.userService.CheckUserPermissionsInDB(
		msg.Context(),       // Context
		msg,                 // Pass the full message
		payload.DiscordID,   // usertypes.DiscordID
		roleEnum,            // usertypes.UserRoleEnum
		payload.RequesterID, // string
	)

	// Handle errors from the service
	if err != nil {
		h.logger.Error("Failed during user permissions check in DB",
			slog.String("correlation_id", correlationID),
			slog.String("discord_id", string(payload.DiscordID)),
			slog.String("role", string(payload.Role)),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed during user permissions check in DB: %w", err)
	}

	h.logger.Info("User permissions check request processed successfully",
		slog.String("correlation_id", correlationID),
		slog.String("discord_id", string(payload.DiscordID)),
		slog.String("role", string(payload.Role)),
		slog.String("requester_id", payload.RequesterID),
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
		slog.String("discord_id", string(payload.DiscordID)),
		slog.String("requester_id", payload.RequesterID),
		slog.String("reason", payload.Reason),
	)

	// Implement logic to handle the failure, e.g.,
	// - Notify the user or admin about the failure
	// - Log details for investigation

	return nil
}
