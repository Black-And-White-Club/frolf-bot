package scorehandlers

import (
	"context"
	"fmt"

	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleBulkCorrectScoreRequest processes a ScoreBulkUpdateRequest by iterating each update
// and invoking the existing CorrectScore service call. It emits success/failure events per user.
func (h *ScoreHandlers) HandleBulkCorrectScoreRequest(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleBulkCorrectScoreRequest",
		&scoreevents.ScoreBulkUpdateRequestPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			bulk, ok := payload.(*scoreevents.ScoreBulkUpdateRequestPayload)
			if !ok {
				return nil, fmt.Errorf("invalid payload type for bulk correct score")
			}
			var out []*message.Message
			var applied, failed int
			var appliedUsers []sharedtypes.DiscordID
			for _, upd := range bulk.Updates {
				res, err := h.scoreService.CorrectScore(ctx, sharedtypes.GuildID(upd.GuildID), upd.RoundID, upd.UserID, upd.Score, upd.TagNumber)
				if err != nil && res.Failure == nil {
					// system error; abort entire batch so it can be retried
					return nil, fmt.Errorf("system error during bulk score update for user %s: %w", upd.UserID, err)
				}
				if res.Failure != nil {
					failed++
					fail, ok := res.Failure.(*scoreevents.ScoreUpdateFailedPayloadV1)
					if ok {
						m, err := h.Helpers.CreateResultMessage(msg, fail, scoreevents.ScoreUpdateFailedV1)
						if err == nil {
							out = append(out, m)
						}
					}
					continue
				}
				if res.Success != nil {
					succ, ok := res.Success.(*scoreevents.ScoreUpdatedPayloadV1)
					if ok {
						m, err := h.Helpers.CreateResultMessage(msg, succ, scoreevents.ScoreUpdatedV1)
						if err == nil {
							out = append(out, m)
						}
						applied++
						appliedUsers = append(appliedUsers, succ.UserID)
					}
				}
			}
			// Add aggregate summary event
			agg := scoreevents.ScoreBulkUpdateSuccessPayload{
				GuildID:        bulk.GuildID,
				RoundID:        bulk.RoundID,
				AppliedCount:   applied,
				FailedCount:    failed,
				TotalRequested: len(bulk.Updates),
				UserIDsApplied: appliedUsers,
			}
			aggMsg, err := h.Helpers.CreateResultMessage(msg, &agg, scoreevents.ScoreBulkUpdatedV1)
			if err == nil {
				out = append(out, aggMsg)
			}

			// Trigger reprocessing if any updates were applied
			if applied > 0 {
				scores, err := h.scoreService.GetScoresForRound(ctx, sharedtypes.GuildID(bulk.GuildID), bulk.RoundID)
				if err != nil {
					h.logger.WarnContext(ctx, "Failed to get scores for reprocessing after bulk score update",
						"round_id", bulk.RoundID,
						"error", err,
					)
				} else if len(scores) > 0 {
					reprocessPayload := scoreevents.ProcessRoundScoresRequestedPayloadV1{
						GuildID: bulk.GuildID,
						RoundID: bulk.RoundID,
						Scores:  scores,
					}
					reprocessMsg, err := h.Helpers.CreateResultMessage(msg, &reprocessPayload, scoreevents.ProcessRoundScoresRequestedV1)
					if err == nil {
						out = append(out, reprocessMsg)
					} else {
						h.logger.WarnContext(ctx, "Failed to create reprocess message after bulk score update",
							"round_id", bulk.RoundID,
							"error", err,
						)
					}
				}
			}

			h.logger.InfoContext(ctx, "Processed bulk score override",
				attr.RoundID("round_id", bulk.RoundID),
				attr.Int("updates_requested", len(bulk.Updates)),
				attr.Int("applied", applied),
				attr.Int("failed", failed),
				attr.Int("emitted_messages", len(out)),
			)
			return out, nil
		},
	)(msg)
}
