package userhandlers

import (
	"context"
	"strings"

	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleTagAvailable handles the TagAvailable event.
func (h *UserHandlers) HandleTagAvailable(
	ctx context.Context,
	payload *sharedevents.TagAvailablePayloadV1,
) ([]handlerwrapper.Result, error) {
	result, err := h.service.CreateUser(ctx, payload.GuildID, payload.UserID, &payload.TagNumber, nil, nil)
	if err != nil {
		return nil, err
	}

	return mapOperationResult(result,
		userevents.UserCreatedV1,
		userevents.UserCreationFailedV1,
	), nil
}

// HandleTagUnavailable handles the TagUnavailable event.
func (h *UserHandlers) HandleTagUnavailable(
	ctx context.Context,
	payload *sharedevents.TagUnavailablePayloadV1,
) ([]handlerwrapper.Result, error) {
	// Ensure a default reason when none is provided
	reason := strings.TrimSpace(payload.Reason)
	if reason == "" {
		reason = "tag not available"
	}

	// Create the UserCreationFailed payload directly - no service call needed
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
