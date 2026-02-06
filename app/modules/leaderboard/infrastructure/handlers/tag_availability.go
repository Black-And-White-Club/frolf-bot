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
	h.logger.Info("HandleTagAvailabilityCheckRequested triggered", "guild_id", payload.GuildID, "user_id", payload.UserID, "tag_number", payload.TagNumber)

	result, err := h.service.CheckTagAvailability(ctx, payload.GuildID, payload.UserID, *payload.TagNumber)
	if err != nil {
		return nil, err
	}
	if result.IsFailure() {
		return nil, *result.Failure
	}

	res := *result.Success
	if res.Available {
		availablePayload := &sharedevents.TagAvailablePayloadV1{
			GuildID:   payload.GuildID,
			UserID:    payload.UserID,
			TagNumber: *payload.TagNumber,
		}

		return []handlerwrapper.Result{{Topic: sharedevents.TagAvailableV1, Payload: availablePayload}}, nil
	}

	tagUnavailable := &sharedevents.TagUnavailablePayloadV1{
		GuildID:   payload.GuildID,
		UserID:    payload.UserID,
		TagNumber: *payload.TagNumber,
		Reason:    res.Reason,
	}

	return []handlerwrapper.Result{{Topic: sharedevents.TagUnavailableV1, Payload: tagUnavailable}}, nil
}
