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
// and invoking the existing CorrectScore service call.
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

	// 1. Process all updates via the service
	for _, upd := range payload.Updates {
		result, err := h.service.CorrectScore(ctx, payload.GuildID, payload.RoundID, upd.UserID, upd.Score, upd.TagNumber)
		if err != nil {
			// Infrastructure error (DB down, etc) - trigger retry/bubbles up
			return nil, err
		}

		if result.Failure != nil {
			// Domain error (Invalid score, Round locked, etc)
			// We cast the Failure (which is an error type) to a string for the message
			return nil, fmt.Errorf("bulk score update failed for user %s: %w", upd.UserID, *result.Failure)
		}
	}

	// 2. Prepare the bulk update event payload
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

	// 3. Wrap result for the handler wrapper
	// Note: Since this is a "Request" handler that triggers further downstream
	// async work, we manually construct the Result slice.
	return []handlerwrapper.Result{{
		Topic:   roundevents.RoundScoreBulkUpdateRequestedV1,
		Payload: bulk,
	}}, nil
}
