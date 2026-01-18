package userhandlers

import (
	"context"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleGetUserRequest handles the GetUserRequest event.
func (h *UserHandlers) HandleGetUserRequest(
	ctx context.Context,
	payload *userevents.GetUserRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	result, err := h.service.GetUser(ctx, payload.GuildID, payload.UserID)
	if err != nil {
		return nil, err
	}

	return mapOperationResult(result,
		userevents.GetUserResponseV1,
		userevents.GetUserFailedV1,
	), nil
}

// HandleGetUserRoleRequest handles the GetUserRoleRequest event.
func (h *UserHandlers) HandleGetUserRoleRequest(
	ctx context.Context,
	payload *userevents.GetUserRoleRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	result, err := h.service.GetUserRole(ctx, payload.GuildID, payload.UserID)
	if err != nil {
		return nil, err
	}

	return mapOperationResult(result,
		userevents.GetUserRoleResponseV1,
		userevents.GetUserRoleFailedV1,
	), nil
}
