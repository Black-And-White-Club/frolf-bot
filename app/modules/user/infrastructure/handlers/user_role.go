package userhandlers

import (
	"fmt"
	"log/slog"

	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	"github.com/Black-And-White-Club/tcr-bot/internal/eventutil"
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
	correlationID, payload, err := eventutil.UnmarshalPayload[userevents.UserPermissionsCheckResponsePayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal UserPermissionsCheckResponse event: %w", err)
	}

	h.logger.Info("Received UserPermissionsCheckResponse event",
		slog.String("correlation_id", correlationID),
		slog.String("user_id", string(payload.DiscordID)),
		slog.String("role", string(payload.Role)),
		slog.Bool("has_permission", payload.HasPermission),
	)

	if payload.HasPermission {
		// Update the user's role in the database
		if err := h.userService.UpdateUserRoleInDatabase(msg.Context(), string(payload.DiscordID), string(payload.Role), correlationID); err != nil {
			// If updating the role fails, publish a UserRoleUpdateFailed event
			if pubErr := h.userService.PublishUserRoleUpdateFailed(msg.Context(), msg, string(payload.DiscordID), string(payload.Role), err.Error()); pubErr != nil {
				return fmt.Errorf("failed to update user role in database and publish UserRoleUpdateFailed event: %w", pubErr)
			}
			return fmt.Errorf("failed to update user role in database: %w", err)
		}
		// If updating the role succeeds, publish a UserRoleUpdated event
		return h.userService.PublishUserRoleUpdated(msg.Context(), msg, string(payload.DiscordID), string(payload.Role))
	} else {
		// If the user does not have permission, publish a UserRoleUpdateFailed event
		return h.userService.PublishUserRoleUpdateFailed(msg.Context(), msg, string(payload.DiscordID), string(payload.Role), "User does not have required permission")
	}
}

// HandleUserRoleUpdateFailed handles the UserRoleUpdateFailed event.
func (h *UserHandlers) HandleUserRoleUpdateFailed(msg *message.Message) error {
	correlationID, payload, err := eventutil.UnmarshalPayload[userevents.UserRoleUpdateFailedPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal UserRoleUpdateFailed event: %w", err)
	}

	h.logger.Error("User role update failed",
		slog.String("correlation_id", correlationID),
		slog.String("user_id", payload.DiscordID),
		slog.String("role", payload.Role),
		slog.String("reason", payload.Reason),
	)

	// Implement logic to handle the failure, e.g.,
	// - Notify the user or admin about the failure
	// - Log details for investigation

	return nil
}
