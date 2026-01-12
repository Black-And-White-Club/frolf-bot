package roundhandlers

import (
	"context"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleScheduledRoundTagUpdate processes tag updates for rounds that are currently in a scheduled state.
// This is triggered when the leaderboard service emits a change in player tags.
func (h *RoundHandlers) HandleScheduledRoundTagUpdate(
	ctx context.Context,
	payload *leaderboardevents.TagUpdateForScheduledRoundsPayloadV1,
) ([]handlerwrapper.Result, error) {
	// If no tags actually changed, there's nothing for the round service to do.
	if len(payload.ChangedTags) == 0 {
		h.logger.DebugContext(ctx, "skipping scheduled round tag update: no changed tags in payload")
		return nil, nil
	}

	// Transform the leaderboard event payload into the format expected by the round service.
	servicePayload := roundevents.ScheduledRoundTagUpdatePayloadV1{
		GuildID:     payload.GuildID,
		ChangedTags: make(map[sharedtypes.DiscordID]*sharedtypes.TagNumber),
	}

	for userID, tag := range payload.ChangedTags {
		// Create a local copy to avoid pointer aliasing during map iteration.
		tagCopy := tag
		servicePayload.ChangedTags[userID] = &tagCopy
	}

	// Execute the update across all eligible scheduled rounds in the database.
	result, err := h.roundService.UpdateScheduledRoundsWithNewTags(ctx, servicePayload)
	if err != nil {
		return nil, err
	}

	// Handle functional failures returned by the service.
	if result.Failure != nil {
		h.logger.WarnContext(ctx, "failed to update scheduled rounds with new tags",
			attr.Any("failure", result.Failure),
		)
		return []handlerwrapper.Result{
			{Topic: roundevents.RoundUpdateErrorV1, Payload: result.Failure},
		}, nil
	}

	// Handle successful updates.
	if result.Success != nil {
		success, ok := result.Success.(*roundevents.TagsUpdatedForScheduledRoundsPayloadV1)
		if !ok {
			return nil, sharedtypes.ValidationError{Message: "unexpected success payload type from UpdateScheduledRoundsWithNewTags"}
		}

		// If the service processed the update but found no rounds actually required a change,
		// we return nil to stop the event chain.
		if len(success.UpdatedRounds) == 0 {
			h.logger.InfoContext(ctx, "scheduled tag update complete: no rounds were affected")
			return nil, nil
		}

		h.logger.InfoContext(ctx, "successfully updated tags for scheduled rounds",
			attr.Int("rounds_updated", len(success.UpdatedRounds)),
			attr.Int("participants_affected", success.Summary.ParticipantsUpdated),
		)

		// Publish the successful update event.
		// This event is consumed by the Discord bot to update the RSVP embeds with the new tag numbers.
		return []handlerwrapper.Result{
			{Topic: roundevents.TagsUpdatedForScheduledRoundsV1, Payload: success},
		}, nil
	}

	return nil, sharedtypes.ValidationError{Message: "unexpected empty result from scheduled tag update service"}
}
