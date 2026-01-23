package roundservice

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
)

// FinalizeRound handles the round finalization process by updating the round state.
func (s *RoundService) FinalizeRound(ctx context.Context, payload roundevents.AllScoresSubmittedPayloadV1) (results.OperationResult, error) {
	return s.withTelemetry(ctx, "FinalizeRound", payload.RoundID, func(ctx context.Context) (results.OperationResult, error) {
		// Update the round state to finalized in the database
		rounddbState := roundtypes.RoundStateFinalized
		s.logger.InfoContext(ctx, "Attempting to update round state to finalized",
			attr.StringUUID("round_id", payload.RoundID.String()),
		)
		if err := s.repo.UpdateRoundState(ctx, payload.GuildID, payload.RoundID, rounddbState); err != nil {
			s.metrics.RecordDBOperationError(ctx, "update_round_state")
			failurePayload := roundevents.RoundFinalizationErrorPayloadV1{
				RoundID: payload.RoundID,
				Error:   fmt.Sprintf("failed to update round state to finalized: %v", err),
			}
			s.logger.ErrorContext(ctx, "Failed to update round state to finalized",
				attr.StringUUID("round_id", payload.RoundID.String()),
				attr.Error(err),
			)
			return results.OperationResult{Failure: &failurePayload}, nil
		}

		// Fetch the finalized round data from the database to get the current state
		round, err := s.repo.GetRound(ctx, payload.GuildID, payload.RoundID)
		if err != nil {
			s.metrics.RecordDBOperationError(ctx, "get_round")
			failurePayload := roundevents.RoundFinalizationErrorPayloadV1{
				RoundID: payload.RoundID,
				Error:   fmt.Sprintf("failed to fetch round data: %v", err),
			}
			s.logger.ErrorContext(ctx, "Failed to fetch round data after finalization",
				attr.StringUUID("round_id", payload.RoundID.String()),
				attr.Error(err),
			)
			return results.OperationResult{Failure: &failurePayload}, nil
		}

		// Prepare the success payload with round data
		finalizedPayload := roundevents.RoundFinalizedPayloadV1{
			GuildID:   payload.GuildID,
			RoundID:   payload.RoundID,
			RoundData: *round,
		}
		s.logger.InfoContext(ctx, "Round state updated to finalized successfully",
			attr.StringUUID("round_id", payload.RoundID.String()),
		)

		return results.OperationResult{Success: &finalizedPayload}, nil
	})
}

// NotifyScoreModule prepares the data needed by the Score Module after a round is finalized.
func (s *RoundService) NotifyScoreModule(ctx context.Context, payload roundevents.RoundFinalizedPayloadV1) (results.OperationResult, error) {

	return s.withTelemetry(ctx, "NotifyScoreModule", payload.RoundID, func(ctx context.Context) (results.OperationResult, error) {
		round := payload.RoundData

		roundMode := sharedtypes.RoundMode("SINGLES")
		if len(round.Teams) > 0 {
			roundMode = sharedtypes.RoundMode("DOUBLES")
		}

		scores := make([]sharedtypes.ScoreInfo, 0, len(round.Participants))
		for _, p := range round.Participants {
			if p.Score == nil {
				continue
			}

			// TagNumbers are meaningful only for singles
			var tagPtr *sharedtypes.TagNumber
			if roundMode == "SINGLES" && p.TagNumber != nil {
				tag := *p.TagNumber
				tagPtr = &tag
			}

			scores = append(scores, sharedtypes.ScoreInfo{
				UserID:    p.UserID,
				TagNumber: tagPtr,
				Score:     *p.Score,
			})
		}

		if len(scores) == 0 {
			return results.OperationResult{
				Failure: &roundevents.RoundFinalizationErrorPayloadV1{
					RoundID: payload.RoundID,
					Error:   "no participants with submitted scores found",
				},
			}, nil
		}

		out := &sharedevents.ProcessRoundScoresRequestedPayloadV1{
			GuildID:      payload.GuildID,
			RoundID:      payload.RoundID,
			Scores:       scores,
			RoundMode:    roundMode,
			Participants: round.Participants, // authoritative grouping source
		}

		s.logger.InfoContext(ctx, "Prepared score processing request",
			attr.StringUUID("round_id", payload.RoundID.String()),
			attr.String("round_mode", string(roundMode)),
			attr.Int("scores", len(scores)),
			attr.Int("participants", len(round.Participants)),
		)

		return results.OperationResult{Success: out}, nil
	})
}

// ConvertToRoundFinalizedPayload converts the event payload to the finalized payload structure.
func ConvertToRoundFinalizedPayload(eventPayload roundevents.AllScoresSubmittedPayloadV1) roundevents.RoundFinalizedPayloadV1 {
	return roundevents.RoundFinalizedPayloadV1{
		RoundID:   eventPayload.RoundID,
		RoundData: eventPayload.RoundData,
	}
}
