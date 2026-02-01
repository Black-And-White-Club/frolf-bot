package userhandlers

import (
	"context"
	"strings"

	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
)

// HandleTagAvailable handles the TagAvailable event.
func (h *UserHandlers) HandleTagAvailable(
	ctx context.Context,
	payload *sharedevents.TagAvailablePayloadV1,
) ([]handlerwrapper.Result, error) {
	if payload == nil {
		return nil, nil // Or return an error depending on your middleware strategy
	}

	// Call the updated service method.
	// Note: udiscUsername and udiscName are passed as nil here
	// as they aren't part of the TagAvailable payload.
	result, err := h.service.CreateUser(
		ctx,
		payload.GuildID,
		payload.UserID,
		&payload.TagNumber,
		nil,
		nil,
	)
	if err != nil {
		// Infrastructure failure (DB down, etc.)
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
			return &userevents.UserCreationFailedPayloadV1{
				GuildID:   payload.GuildID,
				UserID:    payload.UserID,
				TagNumber: &payload.TagNumber,
				Reason:    failure.Error(),
			}
		},
	)

	return mapOperationResult(mappedResult, userevents.UserCreatedV1, userevents.UserCreationFailedV1), nil
}

// HandleTagUnavailable remains largely the same as it doesn't call the service,
// but ensured for consistency.
func (h *UserHandlers) HandleTagUnavailable(
	ctx context.Context,
	payload *sharedevents.TagUnavailablePayloadV1,
) ([]handlerwrapper.Result, error) {
	reason := strings.TrimSpace(payload.Reason)
	if reason == "" {
		reason = "tag not available"
	}

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
