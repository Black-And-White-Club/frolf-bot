package roundservice

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// FinalizeRound handles the round finalization process by updating the round state.
func (s *RoundService) FinalizeRound(ctx context.Context, payload roundevents.AllScoresSubmittedPayloadV1) (RoundOperationResult, error) {
	return s.serviceWrapper(ctx, "FinalizeRound", payload.RoundID, func(ctx context.Context) (RoundOperationResult, error) {
		// Update the round state to finalized in the database
		rounddbState := roundtypes.RoundStateFinalized
		s.logger.InfoContext(ctx, "Attempting to update round state to finalized",
			attr.StringUUID("round_id", payload.RoundID.String()),
		)
		if err := s.RoundDB.UpdateRoundState(ctx, payload.GuildID, payload.RoundID, rounddbState); err != nil {
			s.metrics.RecordDBOperationError(ctx, "update_round_state")
			failurePayload := roundevents.RoundFinalizationErrorPayloadV1{
				RoundID: payload.RoundID,
				Error:   fmt.Sprintf("failed to update round state to finalized: %v", err),
			}
			s.logger.ErrorContext(ctx, "Failed to update round state to finalized",
				attr.StringUUID("round_id", payload.RoundID.String()),
				attr.Error(err),
			)
			return RoundOperationResult{Failure: &failurePayload}, nil
		}

		// Fetch the finalized round data from the database to get the current state
		round, err := s.RoundDB.GetRound(ctx, payload.GuildID, payload.RoundID)
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
			return RoundOperationResult{Failure: &failurePayload}, nil
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

		return RoundOperationResult{Success: &finalizedPayload}, nil
	})
}

// NotifyScoreModule prepares the data needed by the Score Module after a round is finalized.
func (s *RoundService) NotifyScoreModule(ctx context.Context, payload roundevents.RoundFinalizedPayloadV1) (RoundOperationResult, error) {
	return s.serviceWrapper(ctx, "NotifyScoreModule", payload.RoundID, func(ctx context.Context) (RoundOperationResult, error) {
		// Use the round data directly from the payload
		// The payload.RoundData is populated by AllScoresSubmittedPayload which has verified participants with scores
		round := payload.RoundData

		// Prepare the participant score data for the Score Module
		// ONLY include participants who have actually submitted scores
		scores := make([]roundevents.ParticipantScoreV1, 0, len(round.Participants))
		for _, p := range round.Participants {
			// Skip participants without scores
			if p.Score == nil {
				s.logger.DebugContext(ctx, "Skipping participant without score",
					attr.String("user_id", string(p.UserID)),
					attr.StringUUID("round_id", payload.RoundID.String()),
				)
				continue
			}

			tagNumber := 0
			if p.TagNumber != nil && *p.TagNumber != 0 {
				tagNumber = int(*p.TagNumber)
			}

			score := int(*p.Score) // We know p.Score is not nil here

			tagNumberPtr := sharedtypes.TagNumber(tagNumber)
			scores = append(scores, roundevents.ParticipantScoreV1{
				UserID:    sharedtypes.DiscordID(p.UserID),
				TagNumber: &tagNumberPtr,
				Score:     sharedtypes.Score(score),
			})

			s.logger.DebugContext(ctx, "Added participant score",
				attr.String("user_id", string(p.UserID)),
				attr.Int("score", score),
				attr.Int("tag_number", tagNumber),
				attr.StringUUID("round_id", payload.RoundID.String()),
			)
		}

		// Check if we have any scores to process
		if len(scores) == 0 {
			s.logger.WarnContext(ctx, "No participants with scores found for round",
				attr.StringUUID("round_id", payload.RoundID.String()),
				attr.Int("total_participants", len(round.Participants)),
			)
			failurePayload := roundevents.RoundFinalizationErrorPayloadV1{
				RoundID: payload.RoundID,
				Error:   "no participants with submitted scores found",
			}
			return RoundOperationResult{Failure: &failurePayload}, nil
		}

		// Prepare the success payload containing the request for the Score Module
		processScoresPayload := roundevents.ProcessRoundScoresRequestPayloadV1{
			GuildID: payload.GuildID,
			// Use the authoritative RoundID from the event payload to avoid zero UUID writes
			RoundID: payload.RoundID,
			Scores:  scores,
		}
		s.logger.InfoContext(ctx, "Prepared score data for Score Module",
			attr.StringUUID("round_id", payload.RoundID.String()),
			attr.Int("participant_count_processed", len(scores)),
			attr.Int("total_participants", len(round.Participants)),
		)

		return RoundOperationResult{Success: &processScoresPayload}, nil
	})
}

// ConvertToRoundFinalizedPayload converts the event payload to the finalized payload structure.
func ConvertToRoundFinalizedPayload(eventPayload roundevents.AllScoresSubmittedPayloadV1) roundevents.RoundFinalizedPayloadV1 {
	return roundevents.RoundFinalizedPayloadV1{
		RoundID:   eventPayload.RoundID,
		RoundData: eventPayload.RoundData,
	}
}
