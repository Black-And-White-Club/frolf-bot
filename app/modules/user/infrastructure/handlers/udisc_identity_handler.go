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
	result, err := h.service.UpdateUDiscIdentity(
		ctx,
		payload.GuildID,
		payload.UserID,
		payload.Username,
		payload.Name,
	)
	if err != nil {
		return nil, err
	}

	return mapOperationResult(result,
		userevents.UDiscIdentityUpdatedV1,
		userevents.UDiscIdentityUpdateFailedV1,
	), nil
}
