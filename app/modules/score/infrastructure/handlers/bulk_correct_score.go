package scorehandlers

import (
	"context"
	"errors"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleBulkCorrectScoreRequest processes a ScoreBulkUpdateRequest by iterating each update
// and invoking the existing CorrectScore service call. It emits success/failure events per user.
func (h *ScoreHandlers) HandleBulkCorrectScoreRequest(ctx context.Context, payload *sharedevents.ScoreBulkUpdateRequestedPayloadV1) ([]handlerwrapper.Result, error) {
	if payload == nil {
		return nil, errors.New("payload is nil")
	}

	channelID, _ := ctx.Value("channel_id").(string)
	messageID, _ := ctx.Value("message_id").(string)
	if messageID == "" {
		if discordMessageID, ok := ctx.Value("discord_message_id").(string); ok {
			messageID = discordMessageID
		}
	}

	for _, upd := range payload.Updates {
		result, err := h.service.CorrectScore(ctx, payload.GuildID, payload.RoundID, upd.UserID, upd.Score, upd.TagNumber)
		if err != nil {
			return nil, err
		}
		if result.Failure != nil {
			if failure, ok := result.Failure.(*sharedevents.ScoreUpdateFailedPayloadV1); ok {
				return nil, fmt.Errorf("bulk score update failed for user %s: %s", failure.UserID, failure.Reason)
			}
			return nil, fmt.Errorf("bulk score update failed for user %s", upd.UserID)
		}
	}

	updates := make([]roundevents.ScoreUpdateRequestPayloadV1, 0, len(payload.Updates))
	for _, upd := range payload.Updates {
		score := upd.Score
		updates = append(updates, roundevents.ScoreUpdateRequestPayloadV1{
			GuildID:   payload.GuildID,
			RoundID:   payload.RoundID,
			UserID:    upd.UserID,
			Score:     &score,
			ChannelID: channelID,
			MessageID: messageID,
		})
	}

	bulk := &roundevents.ScoreBulkUpdateRequestPayloadV1{
		GuildID:   payload.GuildID,
		RoundID:   payload.RoundID,
		ChannelID: channelID,
		MessageID: messageID,
		Updates:   updates,
	}

	results := []handlerwrapper.Result{{
		Topic:   roundevents.RoundScoreBulkUpdateRequestedV1,
		Payload: bulk,
	}}

	// Handlers delegate observability to the service layer; no logging here.
	return results, nil
}
