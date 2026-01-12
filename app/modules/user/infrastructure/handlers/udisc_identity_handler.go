package userhandlers

import (
	"context"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleUpdateUDiscIdentityRequest handles UDisc identity update requests.
func (h *UserHandlers) HandleUpdateUDiscIdentityRequest(
	ctx context.Context,
	payload *userevents.UpdateUDiscIdentityRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	result, err := h.userService.UpdateUDiscIdentity(
		ctx,
		payload.GuildID,
		payload.UserID,
		payload.Username,
		payload.Name,
	)
	if err != nil {
		return nil, err
	}

	if result.Failure != nil {
		return []handlerwrapper.Result{
			{Topic: userevents.UDiscIdentityUpdateFailedV1, Payload: result.Failure},
		}, nil
	}

	if result.Success != nil {
		return []handlerwrapper.Result{
			{Topic: userevents.UDiscIdentityUpdatedV1, Payload: result.Success},
		}, nil
	}

	return nil, nil
}
