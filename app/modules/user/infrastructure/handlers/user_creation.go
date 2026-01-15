package userhandlers

import (
	"context"
	"errors"

	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleUserSignupRequest handles the UserSignupRequest event.
func (h *UserHandlers) HandleUserSignupRequest(
	ctx context.Context,
	payload *userevents.UserSignupRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	// Check for tag availability
	if payload.TagNumber != nil {
		return []handlerwrapper.Result{
			{Topic: sharedevents.TagAvailabilityCheckRequestedV1, Payload: &sharedevents.TagAvailabilityCheckRequestedPayloadV1{
				GuildID:   payload.GuildID,
				TagNumber: payload.TagNumber,
				UserID:    payload.UserID,
			}},
		}, nil
	}

	// Create user
	result, err := h.userService.CreateUser(ctx, payload.GuildID, payload.UserID, nil, payload.UDiscUsername, payload.UDiscName)
	if err != nil {
		return nil, err
	}

	if result.Failure != nil {
		failedPayload, ok := result.Failure.(*userevents.UserCreationFailedPayloadV1)
		if !ok {
			return nil, errors.New("unexpected type for failure payload")
		}

		return []handlerwrapper.Result{
			{Topic: userevents.UserCreationFailedV1, Payload: failedPayload},
		}, nil
	}

	if result.Success != nil {
		successPayload, ok := result.Success.(*userevents.UserCreatedPayloadV1)
		if !ok {
			return nil, errors.New("unexpected type for success payload")
		}

		return []handlerwrapper.Result{
			{Topic: userevents.UserCreatedV1, Payload: successPayload},
		}, nil
	}

	return nil, errors.New("user creation service returned unexpected result")
}
