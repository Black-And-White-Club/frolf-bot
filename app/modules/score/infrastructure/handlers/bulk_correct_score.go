package scorehandlers

import (
	"context"
	"errors"

	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleBulkCorrectScoreRequest processes a ScoreBulkUpdateRequest by iterating each update
// and invoking the existing CorrectScore service call. It emits success/failure events per user.
func (h *ScoreHandlers) HandleBulkCorrectScoreRequest(ctx context.Context, payload *sharedevents.ScoreBulkUpdateRequestedPayloadV1) ([]handlerwrapper.Result, error) {
	if payload == nil {
		return nil, errors.New("payload is nil")
	}

	var results []handlerwrapper.Result
	var applied, failed int
	var appliedUsers []sharedtypes.DiscordID

	for _, upd := range payload.Updates {
		res, err := h.scoreService.CorrectScore(ctx, sharedtypes.GuildID(upd.GuildID), upd.RoundID, upd.UserID, upd.Score, upd.TagNumber)
		if err != nil && res.Failure == nil {
			// system error; abort entire batch so it can be retried
			return nil, errors.New("system error during bulk score update")
		}
		if res.Failure != nil {
			failed++
			fail, ok := res.Failure.(*sharedevents.ScoreUpdateFailedPayloadV1)
			if ok {
				results = append(results, handlerwrapper.Result{
					Topic:   sharedevents.ScoreUpdateFailedV1,
					Payload: fail,
				})
			}
			continue
		}
		if res.Success != nil {
			succ, ok := res.Success.(*sharedevents.ScoreUpdatedPayloadV1)
			if ok {
				results = append(results, handlerwrapper.Result{
					Topic:   sharedevents.ScoreUpdatedV1,
					Payload: succ,
				})
				applied++
				appliedUsers = append(appliedUsers, succ.UserID)
			}
		}
	}

	// Add aggregate summary event
	agg := &sharedevents.ScoreBulkUpdatedPayloadV1{
		GuildID:        payload.GuildID,
		RoundID:        payload.RoundID,
		AppliedCount:   applied,
		FailedCount:    failed,
		TotalRequested: len(payload.Updates),
		UserIDsApplied: appliedUsers,
	}
	results = append(results, handlerwrapper.Result{
		Topic:   sharedevents.ScoreBulkUpdatedV1,
		Payload: agg,
	})

	// Trigger reprocessing if any updates were applied
	if applied > 0 {
		scores, err := h.scoreService.GetScoresForRound(ctx, sharedtypes.GuildID(payload.GuildID), payload.RoundID)
		if err != nil {
			h.logger.WarnContext(ctx, "Failed to get scores for reprocessing after bulk score update",
				"round_id", payload.RoundID,
				"error", err,
			)
		} else if len(scores) > 0 {
			reprocessPayload := &sharedevents.ProcessRoundScoresRequestedPayloadV1{
				GuildID: payload.GuildID,
				RoundID: payload.RoundID,
				Scores:  scores,
			}
			results = append(results, handlerwrapper.Result{
				Topic:   sharedevents.ProcessRoundScoresRequestedV1,
				Payload: reprocessPayload,
			})
		}
	}

	h.logger.InfoContext(ctx, "Processed bulk score override",
		attr.RoundID("round_id", payload.RoundID),
		attr.Int("updates_requested", len(payload.Updates)),
		attr.Int("applied", applied),
		attr.Int("failed", failed),
		attr.Int("emitted_messages", len(results)),
	)
	return results, nil
}
