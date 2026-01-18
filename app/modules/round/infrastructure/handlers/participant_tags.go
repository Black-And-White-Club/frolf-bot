package roundhandlers

import (
	"context"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleScheduledRoundTagUpdate processes tag updates emitted by the Leaderboard Funnel.
// It synchronizes upcoming rounds with the new source-of-truth tags from the leaderboard.
func (h *RoundHandlers) HandleScheduledRoundTagUpdate(
	ctx context.Context,
	payload *leaderboardevents.TagUpdateForScheduledRoundsPayloadV1,
) ([]handlerwrapper.Result, error) {
	h.logger.InfoContext(ctx, "Processing leaderboard tag update for rounds",
		attr.String("guild_id", string(payload.GuildID)),
		attr.Int("tags_changed", len(payload.ChangedTags)),
	)

	// 1. Pass the map directly to the batch-oriented service method.
	// This supports single swaps, administrative batches, and post-round updates.
	result, err := h.service.UpdateScheduledRoundsWithNewTags(
		ctx,
		payload.GuildID,
		payload.ChangedTags,
	)
	if err != nil {
		h.logger.ErrorContext(ctx, "Internal error during round tag sync", attr.Error(err))
		return nil, err
	}

	// 2. Handle Service-Level Failures and Success
	// Using mapOperationResult to simplify the conversion pattern
	return mapOperationResult(result,
		roundevents.TagsUpdatedForScheduledRoundsV1,
		roundevents.RoundUpdateErrorV1,
	), nil
}
