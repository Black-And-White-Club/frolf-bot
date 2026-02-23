package userhandlers

import (
	"context"
	"strings"

	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	"github.com/google/uuid"
)

// HandleTagAvailable handles the TagAvailable event.
func (h *UserHandlers) HandleTagAvailable(
	ctx context.Context,
	payload *sharedevents.TagAvailablePayloadV1,
) ([]handlerwrapper.Result, error) {
	if payload == nil {
		return nil, nil // Or return an error depending on your middleware strategy
	}

	// Preserve optional UDisc identity captured at signup when tag assignment
	// runs through the availability flow.
	result, err := h.service.CreateUser(
		ctx,
		payload.GuildID,
		payload.UserID,
		&payload.TagNumber,
		payload.UDiscUsername,
		payload.UDiscName,
	)
	if err != nil {
		// Infrastructure failure (DB down, etc.)
		return nil, err
	}

	if result.IsFailure() {
		return []handlerwrapper.Result{
			{
				Topic: userevents.UserCreationFailedV1,
				Payload: &userevents.UserCreationFailedPayloadV1{
					GuildID:   payload.GuildID,
					UserID:    payload.UserID,
					TagNumber: &payload.TagNumber,
					Reason:    (*result.Failure).Error(),
				},
			},
		}, nil
	}

	if !result.IsSuccess() {
		return nil, nil
	}

	success := *result.Success
	events := []handlerwrapper.Result{
		{
			Topic: userevents.UserCreatedV1,
			Payload: &userevents.UserCreatedPayloadV1{
				GuildID:         payload.GuildID,
				UserID:          success.UserID,
				TagNumber:       success.TagNumber,
				IsReturningUser: success.IsReturningUser,
			},
		},
	}

	if success.TagNumber != nil {
		events = append(events, handlerwrapper.Result{
			Topic: sharedevents.LeaderboardBatchTagAssignmentRequestedV1,
			Payload: &sharedevents.BatchTagAssignmentRequestedPayloadV1{
				ScopedGuildID:    sharedevents.ScopedGuildID{GuildID: payload.GuildID},
				RequestingUserID: payload.UserID,
				BatchID:          uuid.NewString(),
				Assignments: []sharedevents.TagAssignmentInfoV1{
					{
						UserID:    payload.UserID,
						TagNumber: *success.TagNumber,
					},
				},
				Source: sharedtypes.ServiceUpdateSourceCreateUser,
			},
		})
	}

	return events, nil
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
