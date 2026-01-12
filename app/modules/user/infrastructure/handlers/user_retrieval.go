package userhandlers

import (
	"context"
	"errors"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleGetUserRequest handles the GetUserRequest event.
func (h *UserHandlers) HandleGetUserRequest(
	ctx context.Context,
	payload *userevents.GetUserRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	result, err := h.userService.GetUser(ctx, payload.GuildID, payload.UserID)
	if err != nil {
		return nil, err
	}

	if result.Failure != nil {
		failedPayload, ok := result.Failure.(*userevents.GetUserFailedPayloadV1)
		if !ok {
			return nil, errors.New("unexpected type for failure payload from GetUser")
		}
		return []handlerwrapper.Result{
			{Topic: userevents.GetUserFailedV1, Payload: failedPayload},
		}, nil
	}

	if result.Success != nil {
		successPayload, ok := result.Success.(*userevents.GetUserResponsePayloadV1)
		if !ok {
			return nil, errors.New("unexpected type for success payload from GetUser")
		}
		return []handlerwrapper.Result{
			{Topic: userevents.GetUserResponseV1, Payload: successPayload},
		}, nil
	}

	return nil, errors.New("get user service returned unexpected result structure")
}

// HandleGetUserRoleRequest handles the GetUserRoleRequest event.
func (h *UserHandlers) HandleGetUserRoleRequest(
	ctx context.Context,
	payload *userevents.GetUserRoleRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	result, err := h.userService.GetUserRole(ctx, payload.GuildID, payload.UserID)
	if err != nil {
		return nil, err
	}

	if result.Failure != nil {
		failedPayload, ok := result.Failure.(*userevents.GetUserRoleFailedPayloadV1)
		if !ok {
			return nil, errors.New("unexpected type for failure payload from GetUserRole")
		}
		return []handlerwrapper.Result{
			{Topic: userevents.GetUserRoleFailedV1, Payload: failedPayload},
		}, nil
	}

	if result.Success != nil {
		successPayload, ok := result.Success.(*userevents.GetUserRoleResponsePayloadV1)
		if !ok {
			return nil, errors.New("unexpected type for success payload from GetUserRole")
		}
		return []handlerwrapper.Result{
			{Topic: userevents.GetUserRoleResponseV1, Payload: successPayload},
		}, nil
	}

	return nil, errors.New("get user role service returned unexpected result structure")
}
