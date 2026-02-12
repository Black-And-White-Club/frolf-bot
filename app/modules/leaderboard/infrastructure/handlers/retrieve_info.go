package leaderboardhandlers

import (
	"context"
	"database/sql"
	"errors"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleGetLeaderboardRequest returns the full current state.
func (h *LeaderboardHandlers) HandleGetLeaderboardRequest(
	ctx context.Context,
	payload *leaderboardevents.GetLeaderboardRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	result, err := h.service.GetLeaderboard(ctx, payload.GuildID, payload.SeasonID)
	if err != nil {
		return []handlerwrapper.Result{{
			Topic: leaderboardevents.GetLeaderboardFailedV1,
			Payload: &leaderboardevents.GetLeaderboardFailedPayloadV1{
				GuildID: payload.GuildID,
				Reason:  err.Error(),
			},
		}}, nil
	}
	if result.IsFailure() {
		return []handlerwrapper.Result{{
			Topic: leaderboardevents.GetLeaderboardFailedV1,
			Payload: &leaderboardevents.GetLeaderboardFailedPayloadV1{
				GuildID: payload.GuildID,
				Reason:  (*result.Failure).Error(),
			},
		}}, nil
	}
	leaderboard := *result.Success

	// Collect user IDs from leaderboard entries
	userIDs := make([]sharedtypes.DiscordID, 0, len(leaderboard))
	for _, entry := range leaderboard {
		userIDs = append(userIDs, entry.UserID)
	}

	// Lookup profiles
	profiles := make(map[sharedtypes.DiscordID]*usertypes.UserProfile)
	var syncRequests []*userevents.UserProfileSyncRequestPayloadV1

	if len(userIDs) > 0 {
		profileResult, _ := h.userService.LookupProfiles(ctx, userIDs, payload.GuildID)
		if profileResult.IsSuccess() {
			resp := profileResult.Success
			profiles = (*resp).Profiles
			syncRequests = (*resp).SyncRequests
		}
	}

	resp := &leaderboardevents.GetLeaderboardResponsePayloadV1{
		GuildID:     payload.GuildID,
		Leaderboard: leaderboard,
		Profiles:    profiles,
	}

	// Check for reply_to subject for Request-Reply pattern
	topic := leaderboardevents.GetLeaderboardResponseV1
	if replyTo, ok := ctx.Value(handlerwrapper.CtxKeyReplyTo).(string); ok && replyTo != "" {
		topic = replyTo
	}

	results := []handlerwrapper.Result{{Topic: topic, Payload: resp}}

	for _, syncReq := range syncRequests {
		results = append(results, handlerwrapper.Result{
			Topic:   userevents.UserProfileSyncRequestTopicV1,
			Payload: syncReq,
		})
	}

	return results, nil
}

// HandleGetTagByUserIDRequest performs a single tag lookup.
func (h *LeaderboardHandlers) HandleGetTagByUserIDRequest(
	ctx context.Context,
	payload *sharedevents.DiscordTagLookupRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	result, err := h.service.GetTagByUserID(ctx, payload.GuildID, payload.UserID)
	found := err == nil && result.IsSuccess()

	var tagPtr *sharedtypes.TagNumber
	if found {
		tagPtr = result.Success
	}

	successPayload := &sharedevents.DiscordTagLookupResultPayloadV1{
		ScopedGuildID:    sharedevents.ScopedGuildID{GuildID: payload.GuildID},
		RequestingUserID: payload.RequestingUserID,
		UserID:           payload.UserID,
		TagNumber:        tagPtr,
		Found:            found,
	}

	topic := sharedevents.LeaderboardTagLookupSucceededV1
	if !found {
		topic = sharedevents.LeaderboardTagLookupNotFoundV1
	}

	return []handlerwrapper.Result{{Topic: topic, Payload: successPayload}}, nil
}

// HandleRoundGetTagRequest handles specialized tag lookups for the Round module.
func (h *LeaderboardHandlers) HandleRoundGetTagRequest(
	ctx context.Context,
	payload *sharedevents.RoundTagLookupRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	result, err := h.service.RoundGetTagByUserID(ctx, payload.GuildID, payload.UserID)

	if err != nil {
		return nil, err
	}
	if result.IsFailure() {
		// Not found -> NotFound event
		if errors.Is(*result.Failure, sql.ErrNoRows) {
			p := &sharedevents.RoundTagLookupResultPayloadV1{
				ScopedGuildID:      sharedevents.ScopedGuildID{GuildID: payload.GuildID},
				UserID:             payload.UserID,
				RoundID:            payload.RoundID,
				TagNumber:          nil,
				Found:              false,
				OriginalResponse:   payload.Response,
				OriginalJoinedLate: payload.JoinedLate,
			}
			return []handlerwrapper.Result{{Topic: sharedevents.RoundTagLookupNotFoundV1, Payload: p}}, nil
		}
		return nil, *result.Failure
	}

	p := &sharedevents.RoundTagLookupResultPayloadV1{
		ScopedGuildID:      sharedevents.ScopedGuildID{GuildID: payload.GuildID},
		UserID:             payload.UserID,
		RoundID:            payload.RoundID,
		TagNumber:          result.Success,
		Found:              true,
		OriginalResponse:   payload.Response,
		OriginalJoinedLate: payload.JoinedLate,
	}
	return []handlerwrapper.Result{{Topic: sharedevents.RoundTagLookupFoundV1, Payload: p}}, nil
}
