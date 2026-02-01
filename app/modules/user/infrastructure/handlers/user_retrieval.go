package userhandlers

import (
	"context"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
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
	mappedResult := result.Map(
		func(success *userservice.UserWithMembership) any {
			return &userevents.GetUserResponsePayloadV1{
				GuildID: payload.GuildID,
				User:    &success.UserData,
			}
		},
		func(failure error) any {
			return &userevents.GetUserFailedPayloadV1{
				GuildID: payload.GuildID,
				UserID:  payload.UserID,
				Reason:  failure.Error(),
			}
		},
	)

	return mapOperationResult(mappedResult, userevents.GetUserResponseV1, userevents.GetUserFailedV1), nil
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

	mappedResult := result.Map(
		func(role sharedtypes.UserRoleEnum) any {
			return &userevents.GetUserRoleResponsePayloadV1{
				GuildID: payload.GuildID,
				UserID:  payload.UserID,
				Role:    role,
			}
		},
		func(failure error) any {
			return &userevents.GetUserRoleFailedPayloadV1{
				GuildID: payload.GuildID,
				UserID:  payload.UserID,
				Reason:  failure.Error(),
			}
		},
	)

	return mapOperationResult(mappedResult, userevents.GetUserRoleResponseV1, userevents.GetUserRoleFailedV1), nil
}
