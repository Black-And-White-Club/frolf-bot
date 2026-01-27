package leaderboardhandlers

import (
	"context"
	"database/sql"
	"errors"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleGetLeaderboardRequest returns the full current state.
func (h *LeaderboardHandlers) HandleGetLeaderboardRequest(
	ctx context.Context,
	payload *leaderboardevents.GetLeaderboardRequestedPayloadV1,
) ([]handlerwrapper.Result, error) {
	leaderboard, err := h.service.GetLeaderboard(ctx, payload.GuildID)
	if err != nil {
		return []handlerwrapper.Result{{
			Topic: leaderboardevents.GetLeaderboardFailedV1,
			Payload: &leaderboardevents.GetLeaderboardFailedPayloadV1{
				GuildID: payload.GuildID,
				Reason:  err.Error(),
			},
		}}, nil
	}

	resp := &leaderboardevents.GetLeaderboardResponsePayloadV1{
		GuildID:     payload.GuildID,
		Leaderboard: leaderboard,
	}

	return []handlerwrapper.Result{{Topic: leaderboardevents.GetLeaderboardResponseV1, Payload: resp}}, nil
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
	tag, err := h.service.RoundGetTagByUserID(ctx, payload.GuildID, payload.UserID)

	if err != nil {
		// Not found -> NotFound event
		if errors.Is(err, sql.ErrNoRows) {
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
		return nil, err
	}

	p := &sharedevents.RoundTagLookupResultPayloadV1{
		ScopedGuildID:      sharedevents.ScopedGuildID{GuildID: payload.GuildID},
		UserID:             payload.UserID,
		RoundID:            payload.RoundID,
		TagNumber:          &tag,
		Found:              true,
		OriginalResponse:   payload.Response,
		OriginalJoinedLate: payload.JoinedLate,
	}
	return []handlerwrapper.Result{{Topic: sharedevents.RoundTagLookupFoundV1, Payload: p}}, nil
}
