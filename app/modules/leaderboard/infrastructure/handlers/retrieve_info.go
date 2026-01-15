package leaderboardhandlers

import (
	"context"
	"time"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
)

// HandleGetLeaderboardRequest returns the full current state.
func (h *LeaderboardHandlers) HandleGetLeaderboardRequest(
	ctx context.Context,
	payload *leaderboardevents.GetLeaderboardRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	result, err := h.leaderboardService.GetLeaderboard(ctx, payload.GuildID)
	if err != nil {
		return nil, err
	}

	leaderboardEntries := make([]leaderboardtypes.LeaderboardEntry, len(result.Leaderboard))
	for i, entry := range result.Leaderboard {
		leaderboardEntries[i] = leaderboardtypes.LeaderboardEntry{
			UserID:    entry.UserID,
			TagNumber: entry.TagNumber,
		}
	}

	return []handlerwrapper.Result{
		{
			Topic: leaderboardevents.GetLeaderboardResponseV1,
			Payload: &leaderboardevents.GetLeaderboardResponsePayloadV1{
				GuildID:     payload.GuildID,
				Leaderboard: leaderboardEntries,
			},
		},
	}, nil
}

// HandleGetTagByUserIDRequest performs a single tag lookup.
func (h *LeaderboardHandlers) HandleGetTagByUserIDRequest(
	ctx context.Context,
	payload *sharedevents.DiscordTagLookupRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	tag, err := h.leaderboardService.GetTagByUserID(ctx, payload.GuildID, payload.UserID)
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
	result leaderboardservice.LeaderboardOperationResult,
	source string,
) []handlerwrapper.Result {
	changedTags := make(map[sharedtypes.DiscordID]sharedtypes.TagNumber)
	assignments := make([]leaderboardevents.TagAssignmentInfoV1, len(result.Leaderboard))

	for i, entry := range result.Leaderboard {
		assignments[i] = leaderboardevents.TagAssignmentInfoV1{
			UserID:    entry.UserID,
			TagNumber: entry.TagNumber,
		}
	}

	for _, change := range result.TagChanges {
		changedTags[change.UserID] = *change.NewTag
	}

	return []handlerwrapper.Result{
		{
			Topic: leaderboardevents.LeaderboardBatchTagAssignedV1,
			Payload: &leaderboardevents.LeaderboardBatchTagAssignedPayloadV1{
				GuildID:          guildID,
				RequestingUserID: requestorID,
				BatchID:          batchID,
				AssignmentCount:  len(result.Leaderboard),
				Assignments:      assignments,
			},
		},
		{
			Topic: sharedevents.TagUpdateForScheduledRoundsV1,
			Payload: &leaderboardevents.TagUpdateForScheduledRoundsPayloadV1{
				GuildID:     guildID,
				ChangedTags: changedTags,
				UpdatedAt:   time.Now().UTC(),
				Source:      source,
			},
		},
	}
}

// HandleRoundGetTagRequest handles specialized tag lookups for the Round module.
func (h *LeaderboardHandlers) HandleRoundGetTagRequest(
	ctx context.Context,
	payload *sharedevents.RoundTagLookupRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	h.logger.InfoContext(ctx, "Received RoundTagLookupRequest event",
		attr.ExtractCorrelationID(ctx),
		attr.String("user_id", string(payload.UserID)),
		attr.String("round_id", payload.RoundID.String()),
	)

	// 1. Call specialized Round lookup in the Service
	result, err := h.leaderboardService.RoundGetTagByUserID(ctx, payload.GuildID, *payload)

	// 2. SYSTEM FAILURE (e.g., DB Connection Lost) -> Trigger Watermill Retry
	if err != nil {
		return nil, err
	}

	// 3. DOMAIN FAILURE (e.g., Guild not initialized) -> ACK and send Failure Event
	if result.Err != nil {
		h.logger.WarnContext(ctx, "Round tag lookup domain failure",
			attr.Error(result.Err))

		return []handlerwrapper.Result{
			{
				Topic: sharedevents.RoundTagLookupFailedV1,
				Payload: &sharedevents.RoundTagLookupFailedPayloadV1{
					ScopedGuildID: sharedevents.ScopedGuildID{GuildID: payload.GuildID},
					UserID:        payload.UserID,
					RoundID:       payload.RoundID,
					Reason:        result.Err.Error(),
				},
			},
		}, nil
	}

	// 4. SUCCESS / NOT FOUND (Business Outcomes)
	// If the user is on the leaderboard, result.Leaderboard will contain 1 entry.
	found := len(result.Leaderboard) > 0
	var tagNumber *sharedtypes.TagNumber
	if found {
		val := result.Leaderboard[0].TagNumber
		tagNumber = &val
	}

	responsePayload := &sharedevents.RoundTagLookupResultPayloadV1{
		ScopedGuildID:      sharedevents.ScopedGuildID{GuildID: payload.GuildID},
		UserID:             payload.UserID,
		RoundID:            payload.RoundID,
		OriginalResponse:   payload.Response,
		OriginalJoinedLate: payload.JoinedLate,
		TagNumber:          tagNumber,
		Found:              found,
	}

	topic := sharedevents.RoundTagLookupFoundV1
	if !found {
		topic = sharedevents.RoundTagLookupNotFoundV1
		h.logger.InfoContext(ctx, "Round participant tag not found",
			attr.String("user_id", string(payload.UserID)))
	}

	return []handlerwrapper.Result{
		{Topic: topic, Payload: responsePayload},
	}, nil
}
