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
func (s *RoundService) FinalizeRound(ctx context.Context, payload roundevents.AllScoresSubmittedPayload) (RoundOperationResult, error) {
	return s.serviceWrapper(ctx, "FinalizeRound", payload.RoundID, func(ctx context.Context) (RoundOperationResult, error) {
		// Update the round state to finalized in the database
		rounddbState := roundtypes.RoundStateFinalized
		s.logger.InfoContext(ctx, "Attempting to update round state to finalized",
			attr.StringUUID("round_id", payload.RoundID.String()),
		)
		if err := s.RoundDB.UpdateRoundState(ctx, payload.RoundID, rounddbState); err != nil {
			s.metrics.RecordDBOperationError(ctx, "update_round_state")
			failurePayload := roundevents.RoundFinalizationErrorPayload{
				RoundID: payload.RoundID,
				Error:   fmt.Sprintf("failed to update round state to finalized: %v", err),
			}
			s.logger.ErrorContext(ctx, "Failed to update round state to finalized",
				attr.StringUUID("round_id", payload.RoundID.String()),
				attr.Error(err),
			)
			return RoundOperationResult{Failure: &failurePayload}, nil
		}

		// Fetch the finalized round data
		round, err := s.RoundDB.GetRound(ctx, payload.RoundID)
		if err != nil {
			s.metrics.RecordDBOperationError(ctx, "get_round")
			failurePayload := roundevents.RoundFinalizationErrorPayload{
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
		finalizedPayload := roundevents.RoundFinalizedPayload{
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
func (s *RoundService) NotifyScoreModule(ctx context.Context, payload roundevents.RoundFinalizedPayload) (RoundOperationResult, error) {
	return s.serviceWrapper(ctx, "NotifyScoreModule", payload.RoundID, func(ctx context.Context) (RoundOperationResult, error) {
		// Check if round exists first
		_, err := s.RoundDB.GetRound(ctx, payload.RoundID)
		if err != nil {
			s.logger.WarnContext(ctx, "Round not found for score processing",
				attr.StringUUID("round_id", payload.RoundID.String()),
				attr.Error(err),
			)
			failurePayload := roundevents.RoundFinalizationErrorPayload{
				RoundID: payload.RoundID,
				Error:   fmt.Sprintf("round not found: %v", err),
			}
			return RoundOperationResult{Failure: &failurePayload}, nil
		}

		// Use the round data directly from the payload
		round := payload.RoundData

		// Prepare the participant score data for the Score Module
		scores := make([]roundevents.ParticipantScore, 0, len(round.Participants))
		for _, p := range round.Participants {
			tagNumber := 0
			if p.TagNumber != nil && *p.TagNumber != 0 {
				tagNumber = int(*p.TagNumber)
			}

			score := 0
			if p.Score != nil {
				score = int(*p.Score)
			}

			tagNumberPtr := sharedtypes.TagNumber(tagNumber)
			scores = append(scores, roundevents.ParticipantScore{
				UserID:    sharedtypes.DiscordID(p.UserID),
				TagNumber: &tagNumberPtr,
				Score:     sharedtypes.Score(score),
			})
		}

		// Prepare the success payload containing the request for the Score Module
		processScoresPayload := roundevents.ProcessRoundScoresRequestPayload{
			RoundID: round.ID,
			Scores:  scores,
		}
		s.logger.InfoContext(ctx, "Prepared score data for Score Module",
			attr.StringUUID("round_id", payload.RoundID.String()),
			attr.Int("participant_count_processed", len(scores)),
		)

		return RoundOperationResult{Success: &processScoresPayload}, nil
	})
}

// ConvertToRoundFinalizedPayload converts the event payload to the finalized payload structure.
func ConvertToRoundFinalizedPayload(eventPayload roundevents.AllScoresSubmittedPayload) roundevents.RoundFinalizedPayload {
	return roundevents.RoundFinalizedPayload{
		RoundID:   eventPayload.RoundID,
		RoundData: eventPayload.RoundData,
	}
}
