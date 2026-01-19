package roundhandlers

import (
	"context"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleScheduledRoundTagSync processes tag updates emitted by the Leaderboard Funnel.
// It synchronizes upcoming rounds with the new source-of-truth tags from the leaderboard.
func (h *RoundHandlers) HandleScheduledRoundTagSync(
	ctx context.Context,
	payload *sharedevents.SyncRoundsTagRequestPayloadV1,
) ([]handlerwrapper.Result, error) {
	h.logger.InfoContext(ctx, "Processing leaderboard tag sync request for rounds",
		attr.String("guild_id", string(payload.GuildID)),
		attr.Int("tags_changed", len(payload.ChangedTags)),
	)

	// 1. Update the Round module's database
	result, err := h.service.UpdateScheduledRoundsWithNewTags(
		ctx,
		payload.GuildID,
		payload.ChangedTags,
	)
	if err != nil {
		h.logger.ErrorContext(ctx, "Internal error during round tag sync", attr.Error(err))
		return nil, err
	}

	// 2. Publish the result to trigger Discord updates
	// Success Topic: roundevents.ScheduledRoundsSyncedV1
	return mapOperationResult(result,
		roundevents.ScheduledRoundsSyncedV1,
		roundevents.RoundUpdateErrorV1,
	), nil
}
