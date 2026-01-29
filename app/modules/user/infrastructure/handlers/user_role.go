package userhandlers

import (
	"context"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleUserRoleUpdateRequest handles the UserRoleUpdateRequest event.
func (h *UserHandlers) HandleUserRoleUpdateRequest(
	ctx context.Context,
	payload *userevents.UserRoleUpdateRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	result, err := h.service.UpdateUserRoleInDatabase(ctx, payload.GuildID, payload.UserID, payload.Role)
	if err != nil {
		return nil, err
	}
	if result.IsSuccess() {
		return []handlerwrapper.Result{
			{
				Topic: userevents.UserRoleUpdatedV1,
				Payload: &userevents.UserRoleUpdateResultPayloadV1{
					GuildID: payload.GuildID,
					UserID:  payload.UserID,
					Role:    payload.Role,
					Success: true,
				},
			},
		}, nil
	}

	if result.IsFailure() {
		return []handlerwrapper.Result{
			{
				Topic: userevents.UserRoleUpdateFailedV1,
				Payload: &userevents.UserRoleUpdateFailedPayloadV1{
					GuildID: payload.GuildID,
					UserID:  payload.UserID,
					Role:    payload.Role,
					Reason:  (*result.Failure).Error(),
				},
			},
		}, nil
	}

	return nil, nil
}
