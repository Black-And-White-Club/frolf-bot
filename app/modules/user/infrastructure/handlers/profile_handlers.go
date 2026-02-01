package userhandlers

import (
	"context"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleUserProfileUpdated processes profile update events from Discord bot
func (h *UserHandlers) HandleUserProfileUpdated(
	ctx context.Context,
	payload *userevents.UserProfileUpdatedPayloadV1,
) ([]handlerwrapper.Result, error) {
	h.logger.Debug("Processing user profile update",
		"user_id", payload.UserID,
		"display_name", payload.DisplayName,
	)

	err := h.service.UpdateUserProfile(ctx,
		payload.UserID,
		payload.DisplayName,
		payload.AvatarHash,
	)

	if err != nil {
		// Log but don't fail - profile updates are best-effort
		// User may not exist yet (e.g., interacting before registration)
		h.logger.Warn("Failed to update user profile",
			"error", err,
			"user_id", payload.UserID,
		)
		// Don't return error - this prevents retries for expected failures
		return nil, nil
	}

	h.logger.Debug("User profile updated successfully",
		"user_id", payload.UserID,
	)

	return nil, nil // No output events
}
