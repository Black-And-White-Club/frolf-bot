package userhandlers

import (
	"context"
	"errors"
	"strings"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleTagAvailable handles the TagAvailable event.
func (h *UserHandlers) HandleTagAvailable(
	ctx context.Context,
	payload *userevents.TagAvailablePayloadV1,
) ([]handlerwrapper.Result, error) {
	result, err := h.userService.CreateUser(ctx, payload.GuildID, payload.UserID, &payload.TagNumber, nil, nil)
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

// HandleTagUnavailable handles the TagUnavailable event.
func (h *UserHandlers) HandleTagUnavailable(
	ctx context.Context,
	payload *userevents.TagUnavailablePayloadV1,
) ([]handlerwrapper.Result, error) {
	// Ensure a default reason when none is provided
	reason := strings.TrimSpace(payload.Reason)
	if reason == "" {
		reason = "tag not available"
	}

	// Create the UserCreationFailed payload
	failedPayload := &userevents.UserCreationFailedPayloadV1{
		GuildID:   payload.GuildID,
		UserID:    payload.UserID,
		TagNumber: &payload.TagNumber,
		Reason:    reason,
	}

	return []handlerwrapper.Result{
		{Topic: userevents.UserCreationFailedV1, Payload: failedPayload},
	}, nil
}
