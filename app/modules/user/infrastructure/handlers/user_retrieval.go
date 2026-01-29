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
	if result.IsSuccess() {
		return []handlerwrapper.Result{
			{
				Topic: userevents.GetUserResponseV1,
				Payload: &userevents.GetUserResponsePayloadV1{
					GuildID: payload.GuildID,
					User:    &(*result.Success).UserData,
				},
			},
		}, nil
	}

	if result.IsFailure() {
		return []handlerwrapper.Result{
			{
				Topic: userevents.GetUserFailedV1,
				Payload: &userevents.GetUserFailedPayloadV1{
					GuildID: payload.GuildID,
					UserID:  payload.UserID,
					Reason:  (*result.Failure).Error(),
				},
			},
		}, nil
	}

	return nil, nil
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
	if result.IsSuccess() {
		return []handlerwrapper.Result{
			{
				Topic: userevents.GetUserRoleResponseV1,
				Payload: &userevents.GetUserRoleResponsePayloadV1{
					GuildID: payload.GuildID,
					UserID:  payload.UserID,
					Role:    *result.Success,
				},
			},
		}, nil
	}

	if result.IsFailure() {
		return []handlerwrapper.Result{
			{
				Topic: userevents.GetUserRoleFailedV1,
				Payload: &userevents.GetUserRoleFailedPayloadV1{
					GuildID: payload.GuildID,
					UserID:  payload.UserID,
					Reason:  (*result.Failure).Error(),
				},
			},
		}, nil
	}

	return nil, nil
}
