package userhandlers

import (
	"fmt"
	"log/slog"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	"github.com/Black-And-White-Club/frolf-bot/internal/eventutil"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleGetUserRequest handles the GetUserRequest event.
func (h *UserHandlers) HandleGetUserRequest(msg *message.Message) error {
	correlationID, payload, err := eventutil.UnmarshalPayload[userevents.GetUserRequestPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal GetUserRequest event: %w", err)
	}

	h.logger.Info("Received GetUserRequest event",
		slog.String("correlation_id", correlationID),
		slog.String("discord_id", string(payload.DiscordID)),
	)

	// Call the service function to get the user
	if err = h.userService.GetUser(msg.Context(), msg, usertypes.DiscordID(payload.DiscordID)); err != nil {
		h.logger.Error("Failed to get user",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to get user: %w", err)
	}

	h.logger.Info("GetUserRequest processed",
		slog.String("correlation_id", correlationID),
		slog.String("discord_id", string(payload.DiscordID)),
	)

	return nil
}

// HandleGetUserRoleRequest handles the GetUserRoleRequest event.
func (h *UserHandlers) HandleGetUserRoleRequest(msg *message.Message) error {
	correlationID, payload, err := eventutil.UnmarshalPayload[userevents.GetUserRoleRequestPayload](msg, h.logger)
	if err != nil {
		return fmt.Errorf("failed to unmarshal GetUserRoleRequest event: %w", err)
	}

	h.logger.Info("Received GetUserRoleRequest event",
		slog.String("correlation_id", correlationID),
		slog.String("discord_id", string(payload.DiscordID)),
	)

	// Call the service function to get the user role
	if err = h.userService.GetUserRole(msg.Context(), msg, usertypes.DiscordID(payload.DiscordID)); err != nil {
		h.logger.Error("Failed to get user role",
			slog.String("correlation_id", correlationID),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to get user role: %w", err)
	}

	h.logger.Info("GetUserRoleRequest processed",
		slog.String("correlation_id", correlationID),
		slog.String("discord_id", string(payload.DiscordID)),
	)

	return nil
}
