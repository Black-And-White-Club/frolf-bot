package userhandlers

import (
	"context"
	"errors"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleUserRoleUpdateRequest handles the UserRoleUpdateRequest event.
func (h *UserHandlers) HandleUserRoleUpdateRequest(
	ctx context.Context,
	payload *userevents.UserRoleUpdateRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	result, err := h.userService.UpdateUserRoleInDatabase(ctx, payload.GuildID, payload.UserID, payload.Role)
	if err != nil {
		return nil, err
	}

	if result.Failure != nil {
		failurePayload, ok := result.Failure.(*userevents.UserRoleUpdateResultPayloadV1)
		if !ok {
			return nil, errors.New("unexpected failure payload type from UpdateUserRoleInDatabase")
		}
		return []handlerwrapper.Result{
			{Topic: userevents.UserRoleUpdateFailedV1, Payload: failurePayload},
		}, nil
	}

	if result.Success != nil {
		successPayload, ok := result.Success.(*userevents.UserRoleUpdateResultPayloadV1)
		if !ok {
			return nil, errors.New("unexpected success payload type from UpdateUserRoleInDatabase")
		}
		return []handlerwrapper.Result{
			{Topic: userevents.UserRoleUpdatedV1, Payload: successPayload},
		}, nil
	}

	return nil, errors.New("update user role service returned unexpected result structure")
}
