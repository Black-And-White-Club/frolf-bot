package userhandlers

import (
	"context"
	"errors"

	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
)

// HandleUserSignupRequest handles the UserSignupRequest event.
func (h *UserHandlers) HandleUserSignupRequest(
	ctx context.Context,
	payload *userevents.UserSignupRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	// Check for tag availability first - this is a special flow that doesn't go through the service
	if payload.TagNumber != nil {
		return []handlerwrapper.Result{
			{Topic: sharedevents.TagAvailabilityCheckRequestedV1, Payload: &sharedevents.TagAvailabilityCheckRequestedPayloadV1{
				GuildID:   payload.GuildID,
				TagNumber: payload.TagNumber,
				UserID:    payload.UserID,
			}},
		}, nil
	}

	// Create user without tag
	result, err := h.service.CreateUser(ctx, payload.GuildID, payload.UserID, nil, payload.UDiscUsername, payload.UDiscName)
	if err != nil {
		return nil, err
	}

	// Map result to event payloads
	mappedResult := result.Map(
		func(success *userservice.CreateUserResponse) any {
			return &userevents.UserCreatedPayloadV1{
				GuildID:         payload.GuildID,
				UserID:          success.UserID,
				TagNumber:       success.TagNumber,
				IsReturningUser: success.IsReturningUser,
			}
		},
		func(failure error) any {
			reason := failure.Error()
			if errors.Is(failure, userservice.ErrUserAlreadyExists) {
				reason = "user already exists in this guild"
			}
			return &userevents.UserCreationFailedPayloadV1{
				GuildID:   payload.GuildID,
				UserID:    payload.UserID,
				TagNumber: payload.TagNumber,
				Reason:    reason,
			}
		},
	)

	return mapOperationResult(mappedResult, userevents.UserCreatedV1, userevents.UserCreationFailedV1), nil
}
