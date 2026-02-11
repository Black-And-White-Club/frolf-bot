package scorehandlers

import (
	"context"
	"errors"

	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// reprocessAfterScoreUpdate is the shared logic for triggering a fresh ProcessRoundScoresRequest
// after score overrides so leaderboard tag assignment reruns using the original tag inputs.
func (h *ScoreHandlers) reprocessAfterScoreUpdate(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) ([]handlerwrapper.Result, error) {
	// Get the authoritative stored scores for this round, which include the original tag numbers
	scores, err := h.service.GetScoresForRound(ctx, guildID, roundID)
	if err != nil {
		return nil, errors.New("failed to load stored scores for reprocess")
	}
	if len(scores) == 0 {
		return nil, nil
	}

	return []handlerwrapper.Result{
		{
			Topic: sharedevents.ProcessRoundScoresRequestedV1,
			Payload: &sharedevents.ProcessRoundScoresRequestedPayloadV1{
				GuildID: guildID,
				RoundID: roundID,
				Scores:  scores,
			},
		},
	}, nil
}

// HandleReprocessAfterBulkScoreUpdate triggers reprocessing after a bulk score override.
func (h *ScoreHandlers) HandleReprocessAfterBulkScoreUpdate(ctx context.Context, payload *sharedevents.ScoreBulkUpdatedPayloadV1) ([]handlerwrapper.Result, error) {
	if payload == nil {
		return nil, errors.New("payload is nil")
	}

	// Skip reprocess if nothing actually applied
	if payload.AppliedCount == 0 {
		return nil, nil
	}

	return h.reprocessAfterScoreUpdate(ctx, payload.GuildID, payload.RoundID)
}

// HandleReprocessAfterSingleScoreUpdate triggers reprocessing after a single score override.
func (h *ScoreHandlers) HandleReprocessAfterSingleScoreUpdate(ctx context.Context, payload *sharedevents.ScoreUpdatedPayloadV1) ([]handlerwrapper.Result, error) {
	if payload == nil {
		return nil, errors.New("payload is nil")
	}

	return h.reprocessAfterScoreUpdate(ctx, payload.GuildID, payload.RoundID)
}
