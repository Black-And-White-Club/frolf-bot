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
	mappedResult := result.Map(
		func(_ bool) any {
			return &userevents.UserRoleUpdateResultPayloadV1{
				GuildID: payload.GuildID,
				UserID:  payload.UserID,
				Role:    payload.Role,
				Success: true,
			}
		},
		func(failure error) any {
			return &userevents.UserRoleUpdateFailedPayloadV1{
				GuildID: payload.GuildID,
				UserID:  payload.UserID,
				Role:    payload.Role,
				Reason:  failure.Error(),
			}
		},
	)

	return mapOperationResult(mappedResult, userevents.UserRoleUpdatedV1, userevents.UserRoleUpdateFailedV1), nil
}
