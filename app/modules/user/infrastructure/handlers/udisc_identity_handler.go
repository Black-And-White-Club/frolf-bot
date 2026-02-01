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
	// Service only wants UserID and identity fields
	result, err := h.service.UpdateUDiscIdentity(
		ctx,
		payload.UserID,
		payload.Username,
		payload.Name,
	)
	if err != nil {
		return nil, err
	}

	mappedResult := result.Map(
		func(_ bool) any {
			return &userevents.UDiscIdentityUpdatedPayloadV1{
				GuildID:  payload.GuildID,
				UserID:   payload.UserID,
				Username: payload.Username,
				Name:     payload.Name,
			}
		},
		func(failure error) any {
			return &userevents.UDiscIdentityUpdateFailedPayloadV1{
				GuildID: payload.GuildID,
				UserID:  payload.UserID,
				Reason:  failure.Error(),
			}
		},
	)

	return mapOperationResult(mappedResult,
		userevents.UDiscIdentityUpdatedV1,
		userevents.UDiscIdentityUpdateFailedV1,
	), nil
}
