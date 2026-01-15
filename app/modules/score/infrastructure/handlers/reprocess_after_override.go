package scorehandlers

import (
	"context"
	"errors"

	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleReprocessAfterScoreUpdate triggers a fresh ProcessRoundScoresRequest after score overrides
// so leaderboard tag assignment reruns using the original tag inputs for the round.
//
// This handler accepts interface{} because it handles multiple payload types:
// - ScoreBulkUpdatedPayloadV1: always reprocess
// - ScoreUpdatedPayloadV1: only reprocess if not part of a bulk batch
func (h *ScoreHandlers) HandleReprocessAfterScoreUpdate(ctx context.Context, payload interface{}) ([]handlerwrapper.Result, error) {
	if payload == nil {
		return nil, errors.New("payload is nil")
	}

	var guildID sharedtypes.GuildID
	var roundID sharedtypes.RoundID
	var shouldSkip bool

	// Try to unmarshal as bulk first
	if bulk, ok := payload.(*sharedevents.ScoreBulkUpdatedPayloadV1); ok {
		// Skip reprocess if nothing actually applied
		if bulk.AppliedCount == 0 {
			h.logger.InfoContext(ctx, "Skipping reprocess; bulk override applied zero updates",
				attr.RoundID("round_id", bulk.RoundID), attr.Int("applied_count", bulk.AppliedCount))
			return nil, nil
		}
		guildID = bulk.GuildID
		roundID = bulk.RoundID
	} else if single, ok := payload.(*sharedevents.ScoreUpdatedPayloadV1); ok {
		guildID = single.GuildID
		roundID = single.RoundID
		// Skip if this is part of a bulk override (to prevent double-run)
		// Note: This metadata check would be handled at router level if needed
		shouldSkip = false
	} else {
		return nil, errors.New("unexpected payload type")
	}

	if shouldSkip {
		h.logger.InfoContext(ctx, "Skipping reprocess on single success inside bulk batch")
		return nil, nil
	}

	// Get the authoritative stored scores for this round, which include the original tag numbers
	scores, err := h.scoreService.GetScoresForRound(ctx, guildID, roundID)
	if err != nil {
		return nil, errors.New("failed to load stored scores for reprocess")
	}
	if len(scores) == 0 {
		h.logger.WarnContext(ctx, "No stored scores found for reprocess; skipping",
			attr.RoundID("round_id", roundID),
			attr.String("guild_id", string(guildID)),
		)
		return nil, nil
	}

	// Build and return a ProcessRoundScoresRequest with existing scores
	req := &sharedevents.ProcessRoundScoresRequestedPayloadV1{
		GuildID: guildID,
		RoundID: roundID,
		Scores:  scores,
	}

	h.logger.InfoContext(ctx, "Reprocessing round scores after override",
		attr.RoundID("round_id", roundID), attr.Int("score_count", len(scores)))

	return []handlerwrapper.Result{
		{
			Topic:   sharedevents.ProcessRoundScoresRequestedV1,
			Payload: req,
		},
	}, nil
}
