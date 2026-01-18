package userhandlers

import (
	"context"

	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
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

	return mapOperationResult(result,
		userevents.UserCreatedV1,
		userevents.UserCreationFailedV1,
	), nil
}
