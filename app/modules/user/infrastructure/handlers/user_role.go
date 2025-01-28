package userhandlers

import (
	"fmt"
	"log/slog"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleUserRoleUpdateRequest handles the UserRoleUpdateRequest event.
func (h *UserHandlers) HandleUserRoleUpdateRequest(msg *message.Message) error {
	correlationID, payload, err := eventutil.UnmarshalPayload[userevents.UserRoleUpdateRequestPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal UserRoleUpdateRequest event: %w", err)
	}

	h.logger.Info("Received UserRoleUpdateRequest event",
		slog.String("correlation_id", correlationID),
		slog.String("user_id", string(payload.DiscordID)),
		slog.String("role", string(payload.Role)),
		slog.String("requester_id", payload.RequesterID),
	)

	// Call the service function to start the update process
	if err := h.userService.UpdateUserRole(msg.Context(), msg, payload.DiscordID, payload.Role, payload.RequesterID); err != nil {
		h.logger.Error("Failed to initiate user role update",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to initiate user role update: %w", err)
	}

	h.logger.Info("User role update request processed", slog.String("correlation_id", correlationID))

	return nil
}

// HandleUserPermissionsCheckResponse handles the UserPermissionsCheckResponse event.
func (h *UserHandlers) HandleUserPermissionsCheckResponse(msg *message.Message) error {
	// Extract correlation ID and payload
	correlationID, payload, err := eventutil.UnmarshalPayload[userevents.UserPermissionsCheckResponsePayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal UserPermissionsCheckResponse event: %w", err)
	}

	h.logger.Info("Received UserPermissionsCheckResponse event",
		slog.String("correlation_id", correlationID),
		slog.String("discord_id", string(payload.DiscordID)),
		slog.String("role", string(payload.Role)),
		slog.Bool("has_permission", payload.HasPermission),
	)

	// If the user has permission
	if payload.HasPermission {
		// Attempt to update the user's role in the database
		if err := h.userService.UpdateUserRoleInDatabase(
			msg.Context(),
			msg,
			payload.DiscordID,
			payload.Role,
		); err != nil {
			// If updating the role fails, publish a UserRoleUpdateFailed event
			h.logger.Error("Failed to update user role in database",
				slog.String("correlation_id", correlationID),
				slog.String("discord_id", string(payload.DiscordID)),
				slog.String("role", string(payload.Role)),
				slog.Any("error", err),
			)
			if pubErr := h.userService.PublishUserRoleUpdateFailed(
				msg.Context(),
				msg,
				payload.DiscordID,
				payload.Role,
				err.Error(),
			); pubErr != nil {
				h.logger.Error("Failed to publish UserRoleUpdateFailed event",
					slog.String("correlation_id", correlationID),
					slog.String("discord_id", string(payload.DiscordID)),
					slog.String("role", string(payload.Role)),
					slog.Any("error", pubErr),
				)
				return fmt.Errorf("failed to update user role and publish UserRoleUpdateFailed event: %w", pubErr)
			}
			return fmt.Errorf("failed to update user role in database: %w", err)
		}

		// If updating the role succeeds, publish a UserRoleUpdated event
		h.logger.Info("User role updated successfully",
			slog.String("correlation_id", correlationID),
			slog.String("discord_id", string(payload.DiscordID)),
			slog.String("role", string(payload.Role)),
		)
		return h.userService.PublishUserRoleUpdated(
			msg.Context(),
			msg,
			payload.DiscordID,
			payload.Role,
		)
	}

	// If the user does not have permission, publish a UserRoleUpdateFailed event
	h.logger.Warn("User does not have required permission",
		slog.String("correlation_id", correlationID),
		slog.String("discord_id", string(payload.DiscordID)),
		slog.String("role", string(payload.Role)),
	)
	return h.userService.PublishUserRoleUpdateFailed(
		msg.Context(),
		msg,
		payload.DiscordID,
		payload.Role,
		"User does not have required permission",
	)
}
