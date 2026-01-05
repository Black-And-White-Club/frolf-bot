package scorehandlers

import (
	"context"
	"fmt"

	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleReprocessAfterScoreUpdate triggers a fresh ProcessRoundScoresRequest after score overrides
// so leaderboard tag assignment reruns using the original tag inputs for the round.
//
// Behavior:
//   - For ScoreBulkUpdateSuccess: always fetch current stored round scores and republish ProcessRoundScoresRequest
//   - For ScoreUpdateSuccess: if metadata indicates part of a bulk override (override=true && override_mode=bulk), do nothing here
//     to avoid duplicate reprocess; otherwise, fetch stored round scores and republish to keep logic consistent.
func (h *ScoreHandlers) HandleReprocessAfterScoreUpdate(msg *message.Message) ([]*message.Message, error) {
	handlerName := "HandleReprocessAfterScoreUpdate"
	// Decide payload type based on topic
	topic := msg.Metadata.Get("topic")

	// Wrap with common handler wrapper by providing a dynamic unmarshal target
	// We’ll unmarshal to the superset we need depending on topic.
	return h.handlerWrapper(handlerName, nil, func(ctx context.Context, msg *message.Message, _ interface{}) ([]*message.Message, error) {
		// Determine if this is bulk or single success
		isBulk := topic == scoreevents.ScoreBulkUpdatedV1

		if !isBulk {
			// Single success; skip if part of bulk to prevent double-run
			if msg.Metadata.Get("override") == "true" && msg.Metadata.Get("override_mode") == "bulk" {
				h.logger.InfoContext(ctx, "Skipping reprocess on single success inside bulk batch")
				return nil, nil
			}
		}

		var guildID sharedtypes.GuildID
		var roundID sharedtypes.RoundID

		if isBulk {
			var p scoreevents.ScoreBulkUpdatedPayloadV1
			if err := h.Helpers.UnmarshalPayload(msg, &p); err != nil {
				return nil, fmt.Errorf("failed to unmarshal bulk success payload: %w", err)
			}
			// Skip reprocess if nothing actually applied
			if p.AppliedCount == 0 {
				h.logger.InfoContext(ctx, "Skipping reprocess; bulk override applied zero updates",
					attr.RoundID("round_id", p.RoundID), attr.Int("applied_count", p.AppliedCount))
				return nil, nil
			}
			guildID = p.GuildID
			roundID = p.RoundID
		} else {
			var p scoreevents.ScoreUpdatedPayloadV1
			if err := h.Helpers.UnmarshalPayload(msg, &p); err != nil {
				return nil, fmt.Errorf("failed to unmarshal success payload: %w", err)
			}
			guildID = p.GuildID
			roundID = p.RoundID
		}

		// Get the authoritative stored scores for this round, which include the original tag numbers
		scores, err := h.scoreService.GetScoresForRound(ctx, guildID, roundID)
		if err != nil {
			return nil, fmt.Errorf("failed to load stored scores for reprocess: %w", err)
		}
		if len(scores) == 0 {
			h.logger.WarnContext(ctx, "No stored scores found for reprocess; skipping",
				attr.RoundID("round_id", roundID),
				attr.String("guild_id", string(guildID)),
			)
			return nil, nil
		}

		// Build and publish a ProcessRoundScoresRequest with existing scores
		req := scoreevents.ProcessRoundScoresRequestedPayloadV1{
			GuildID: guildID,
			RoundID: roundID,
			Scores:  scores,
		}
		out, err := h.Helpers.CreateResultMessage(msg, &req, scoreevents.ProcessRoundScoresRequestedV1)
		if err != nil {
			return nil, fmt.Errorf("failed to create reprocess request message: %w", err)
		}
		// Ensure routing metadata present
		out.Metadata.Set("topic", scoreevents.ProcessRoundScoresRequestedV1)
		h.logger.InfoContext(ctx, "Published reprocess request after score override",
			attr.RoundID("round_id", roundID), attr.Int("score_count", len(scores)))
		return []*message.Message{out}, nil
	})(msg)
}
