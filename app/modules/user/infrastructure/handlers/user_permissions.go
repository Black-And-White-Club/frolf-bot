package userhandlers

import (
	"fmt"
	"log/slog"

	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
	usertypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types"
	"github.com/Black-And-White-Club/tcr-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
)

func (h *UserHandlers) HandleUserPermissionsCheckRequest(msg *message.Message) error {
	correlationID, payload, err := eventutil.UnmarshalPayload[userevents.UserPermissionsCheckRequestPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal UserPermissionsCheckRequest event: %w", err)
	}

	h.logger.Info("Received UserPermissionsCheckRequest event",
		slog.String("correlation_id", correlationID),
		slog.String("user_id", string(payload.DiscordID)),
		slog.String("role", payload.Role),
		slog.String("requester_id", payload.RequesterID),
	)

	// Call the CheckUserPermissionsInDB function to validate permissions
	err = h.userService.CheckUserPermissionsInDB(
		msg.Context(),
		payload.DiscordID,
		usertypes.UserRoleEnum(payload.Role),
		payload.RequesterID,
		correlationID,
	)
	if err != nil {
		h.logger.Error("Failed during user permissions check in DB",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed during user permissions check in DB: %w", err)
	}

	h.logger.Info("User permissions check request processed", slog.String("correlation_id", correlationID))

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
		slog.String("role", payload.Role),
		slog.String("requester_id", payload.RequesterID),
		slog.String("reason", payload.Reason),
	)

	// Implement logic to handle the failure, e.g.,
	// - Notify the user or admin about the failure
	// - Log details for investigation

	return nil
}
