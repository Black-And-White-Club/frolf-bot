package leaderboardhandlers

import (
	"context"
	"log/slog"
	"time"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleTagHistoryRequest processes a tag history request-reply.
func (h *LeaderboardHandlers) HandleTagHistoryRequest(
	ctx context.Context,
	payload *leaderboardevents.TagHistoryRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	h.logger.DebugContext(ctx, "tag history requested",
		slog.String("guild_id", payload.GuildID),
		slog.String("member_id", payload.MemberID),
		slog.Int("limit", payload.Limit),
	)
	history, err := h.service.GetTagHistory(ctx, sharedtypes.GuildID(payload.GuildID), payload.MemberID, payload.Limit)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to get tag history",
			slog.String("guild_id", payload.GuildID),
			slog.String("member_id", payload.MemberID),
			slog.String("error", err.Error()),
		)
		return []handlerwrapper.Result{{
			Topic: leaderboardevents.LeaderboardTagHistoryFailedV1,
			Payload: &leaderboardevents.TagHistoryFailedPayloadV1{
				GuildID: payload.GuildID,
				Reason:  "unable to retrieve tag history",
			},
		}}, nil
	}

	entries := make([]leaderboardevents.TagHistoryEntryV1, len(history))
	for i, entry := range history {
		entries[i] = leaderboardevents.TagHistoryEntryV1{
			ID:          entry.ID,
			TagNumber:   entry.TagNumber,
			OldMemberID: entry.OldMemberID,
			NewMemberID: entry.NewMemberID,
			Reason:      entry.Reason,
			CreatedAt:   entry.CreatedAt.Format(time.RFC3339),
		}
		if entry.RoundID != nil {
			entries[i].RoundID = *entry.RoundID
		}
	}

	topic := leaderboardevents.LeaderboardTagHistoryResponseV1
	if replyTo, ok := ctx.Value(handlerwrapper.CtxKeyReplyTo).(string); ok && replyTo != "" {
		topic = replyTo
	}

	return []handlerwrapper.Result{{
		Topic: topic,
		Payload: &leaderboardevents.TagHistoryResponsePayloadV1{
			GuildID: payload.GuildID,
			Entries: entries,
		},
	}}, nil
}

// HandleTagGraphRequest processes a tag graph PNG request-reply.
func (h *LeaderboardHandlers) HandleTagGraphRequest(
	ctx context.Context,
	payload *leaderboardevents.TagGraphRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	pngData, err := h.service.GenerateTagGraphPNG(ctx, sharedtypes.GuildID(payload.GuildID), payload.MemberID)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to generate tag graph",
			slog.String("guild_id", payload.GuildID),
			slog.String("member_id", payload.MemberID),
			slog.String("error", err.Error()),
		)
		return []handlerwrapper.Result{{
			Topic: leaderboardevents.LeaderboardTagGraphFailedV1,
			Payload: &leaderboardevents.TagGraphFailedPayloadV1{
				GuildID:  payload.GuildID,
				MemberID: payload.MemberID,
				Reason:   "unable to generate tag graph",
			},
		}}, nil
	}

	topic := leaderboardevents.LeaderboardTagGraphResponseV1
	if replyTo, ok := ctx.Value(handlerwrapper.CtxKeyReplyTo).(string); ok && replyTo != "" {
		topic = replyTo
	}

	return []handlerwrapper.Result{{
		Topic: topic,
		Payload: &leaderboardevents.TagGraphResponsePayloadV1{
			GuildID:  payload.GuildID,
			MemberID: payload.MemberID,
			PNGData:  pngData,
		},
	}}, nil
}

// HandleTagListRequest processes a tag list request-reply for the PWA master list.
func (h *LeaderboardHandlers) HandleTagListRequest(
	ctx context.Context,
	payload *leaderboardevents.TagListRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	tagList, err := h.service.GetTagList(ctx, sharedtypes.GuildID(payload.GuildID), payload.ClubUUID)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to get tag list",
			slog.String("guild_id", payload.GuildID),
			slog.String("error", err.Error()),
		)
		return []handlerwrapper.Result{{
			Topic: leaderboardevents.LeaderboardTagListFailedV1,
			Payload: &leaderboardevents.TagListFailedPayloadV1{
				GuildID: payload.GuildID,
				Reason:  "unable to retrieve tag list",
			},
		}}, nil
	}

	members := make([]leaderboardevents.TagListMemberV1, len(tagList))
	userIDs := make([]sharedtypes.DiscordID, 0, len(tagList))
	for i, m := range tagList {
		members[i] = leaderboardevents.TagListMemberV1{
			MemberID:   m.MemberID,
			CurrentTag: m.Tag,
		}
		userIDs = append(userIDs, sharedtypes.DiscordID(m.MemberID))
	}

	profiles := make(map[sharedtypes.DiscordID]*usertypes.UserProfile)
	var syncRequests []*userevents.UserProfileSyncRequestPayloadV1

	if len(userIDs) > 0 {
		profileResult, _ := h.userService.LookupProfiles(ctx, userIDs, sharedtypes.GuildID(payload.GuildID))
		if profileResult.IsSuccess() {
			resp := profileResult.Success
			profiles = (*resp).Profiles
			syncRequests = (*resp).SyncRequests
		}
	}

	topic := leaderboardevents.LeaderboardTagListResponseV1
	if replyTo, ok := ctx.Value(handlerwrapper.CtxKeyReplyTo).(string); ok && replyTo != "" {
		topic = replyTo
	}

	results := make([]handlerwrapper.Result, 0, 1+len(syncRequests))
	results = append(results, handlerwrapper.Result{
		Topic: topic,
		Payload: &leaderboardevents.TagListResponsePayloadV1{
			GuildID:  payload.GuildID,
			Members:  members,
			Profiles: profiles,
		},
	})

	for _, syncReq := range syncRequests {
		results = append(results, handlerwrapper.Result{
			Topic:   userevents.UserProfileSyncRequestTopicV1,
			Payload: syncReq,
		})
	}

	return results, nil
}
