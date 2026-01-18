package leaderboardhandlers

import (
	"context"

	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleTagAvailabilityCheckRequested checks tag availability and emits the appropriate outcome event.
func (h *LeaderboardHandlers) HandleTagAvailabilityCheckRequested(
	ctx context.Context,
	payload *sharedevents.TagAvailabilityCheckRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	result, failure, err := h.service.CheckTagAvailability(ctx, payload.GuildID, payload.UserID, payload.TagNumber)
	if err != nil {
		return nil, err
	}

	if failure != nil {
		return []handlerwrapper.Result{
			{Topic: sharedevents.TagAvailabilityCheckFailedV1, Payload: failure},
		}, nil
	}

	if result.Available {
		availablePayload := &sharedevents.TagAvailablePayloadV1{
			GuildID:   result.GuildID,
			UserID:    result.UserID,
			TagNumber: *result.TagNumber,
		}

		return []handlerwrapper.Result{
			{Topic: sharedevents.TagAvailableV1, Payload: availablePayload},
		}, nil
	}

	tagUnavailable := &sharedevents.TagUnavailablePayloadV1{
		GuildID:   result.GuildID,
		UserID:    result.UserID,
		TagNumber: *result.TagNumber,
		Reason:    result.Reason,
	}

	return []handlerwrapper.Result{
		{Topic: sharedevents.TagUnavailableV1, Payload: tagUnavailable},
	}, nil
}
