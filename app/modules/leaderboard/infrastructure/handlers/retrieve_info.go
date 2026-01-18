package leaderboardhandlers

import (
	"context"
	"fmt"
	"time"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
)

// HandleGetLeaderboardRequest returns the full current state.
func (h *LeaderboardHandlers) HandleGetLeaderboardRequest(
	ctx context.Context,
	payload *leaderboardevents.GetLeaderboardRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	result, err := h.service.GetLeaderboard(ctx, payload.GuildID)
	if err != nil {
		return nil, err
	}

	return mapOperationResult(result,
		leaderboardevents.GetLeaderboardResponseV1,
		leaderboardevents.GetLeaderboardFailedV1,
	), nil
}

// HandleGetTagByUserIDRequest performs a single tag lookup.
func (h *LeaderboardHandlers) HandleGetTagByUserIDRequest(
	ctx context.Context,
	payload *sharedevents.DiscordTagLookupRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	tag, err := h.service.GetTagByUserID(ctx, payload.GuildID, payload.UserID)
	found := err == nil

	var tagPtr *sharedtypes.TagNumber
	if found {
		tagPtr = &tag
	}

	successPayload := &sharedevents.DiscordTagLookupResultPayloadV1{
		ScopedGuildID: sharedevents.ScopedGuildID{GuildID: payload.GuildID},
		UserID:        payload.UserID,
		TagNumber:     tagPtr,
		Found:         found,
	}

	topic := sharedevents.LeaderboardTagLookupSucceededV1
	if !found {
		topic = sharedevents.LeaderboardTagLookupNotFoundV1
	}

	return []handlerwrapper.Result{{Topic: topic, Payload: successPayload}}, nil
}

// mapSuccessResults is a private helper to build consistent batch completion events.
func (h *LeaderboardHandlers) mapSuccessResults(
	guildID sharedtypes.GuildID,
	requestorID sharedtypes.DiscordID,
	batchID string,
	result results.OperationResult,
	source string,
) []handlerwrapper.Result {
	// Expect the service to return a LeaderboardBatchTagAssignedPayloadV1
	var assignments []leaderboardevents.TagAssignmentInfoV1
	if result.IsSuccess() {
		if payload, ok := result.Success.(*leaderboardevents.LeaderboardBatchTagAssignedPayloadV1); ok {
			assignments = payload.Assignments
		}
	}

	changedTags := make(map[sharedtypes.DiscordID]sharedtypes.TagNumber)
	for _, a := range assignments {
		changedTags[a.UserID] = a.TagNumber
	}

	// Build the batch-assigned payload using the handler-provided metadata
	batchPayload := &leaderboardevents.LeaderboardBatchTagAssignedPayloadV1{
		GuildID:          guildID,
		RequestingUserID: requestorID,
		BatchID:          batchID,
		AssignmentCount:  len(assignments),
		Assignments:      assignments,
	}

	return []handlerwrapper.Result{
		{Topic: leaderboardevents.LeaderboardBatchTagAssignedV1, Payload: batchPayload},
		{Topic: sharedevents.TagUpdateForScheduledRoundsV1, Payload: &leaderboardevents.TagUpdateForScheduledRoundsPayloadV1{
			GuildID:     guildID,
			ChangedTags: changedTags,
			UpdatedAt:   time.Now().UTC(),
			Source:      source,
		}},
	}
}

// HandleRoundGetTagRequest handles specialized tag lookups for the Round module.
func (h *LeaderboardHandlers) HandleRoundGetTagRequest(
	ctx context.Context,
	payload *sharedevents.RoundTagLookupRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	// 1. Call specialized Round lookup in the Service
	result, err := h.service.RoundGetTagByUserID(ctx, payload.GuildID, *payload)

	// 2. SYSTEM FAILURE (e.g., DB Connection Lost) -> Trigger Watermill Retry
	if err != nil {
		return nil, err
	}

	// 3. DOMAIN FAILURE -> ACK and send Failure Event (single-topic behavior retained)
	if result.IsFailure() {
		var reason string
		if p, ok := result.Failure.(*sharedevents.RoundTagLookupFailedPayloadV1); ok {
			reason = p.Reason
		} else {
			reason = fmt.Sprintf("%v", result.Failure)
		}

		return []handlerwrapper.Result{
			{
				Topic: sharedevents.RoundTagLookupFailedV1,
				Payload: &sharedevents.RoundTagLookupFailedPayloadV1{
					ScopedGuildID: sharedevents.ScopedGuildID{GuildID: payload.GuildID},
					UserID:        payload.UserID,
					RoundID:       payload.RoundID,
					Reason:        reason,
				},
			},
		}, nil
	}

	// 4. SUCCESS Path: Expect a RoundTagLookupResultPayloadV1
	if result.IsSuccess() {
		if p, ok := result.Success.(*sharedevents.RoundTagLookupResultPayloadV1); ok {
			topic := sharedevents.RoundTagLookupFoundV1
			if !p.Found {
				topic = sharedevents.RoundTagLookupNotFoundV1
			}
			return []handlerwrapper.Result{{Topic: topic, Payload: p}}, nil
		}
		// Unexpected success payload
		return nil, fmt.Errorf("unexpected success payload type: %T", result.Success)
	}

	// All success/failure paths handled above; should not reach here.
	return nil, nil
}
