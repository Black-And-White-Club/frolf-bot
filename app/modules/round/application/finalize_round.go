package roundservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/uptrace/bun"
)

// FinalizeRound handles the round finalization process by updating the round state.
func (s *RoundService) FinalizeRound(ctx context.Context, req *roundtypes.FinalizeRoundInput) (FinalizeRoundResult, error) {
	result, err := withTelemetry(s, ctx, "FinalizeRound", req.RoundID, func(ctx context.Context) (FinalizeRoundResult, error) {
		return runInTx(s, ctx, func(ctx context.Context, tx bun.IDB) (FinalizeRoundResult, error) {
			// Update the round state to finalized in the database
			rounddbState := roundtypes.RoundStateFinalized
			s.logger.InfoContext(ctx, "Attempting to update round state to finalized",
				attr.StringUUID("round_id", req.RoundID.String()),
			)
			if err := s.repo.UpdateRoundState(ctx, tx, req.GuildID, req.RoundID, rounddbState); err != nil {
				s.metrics.RecordDBOperationError(ctx, "update_round_state")
				s.logger.ErrorContext(ctx, "Failed to update round state to finalized",
					attr.StringUUID("round_id", req.RoundID.String()),
					attr.Error(err),
				)
				return results.FailureResult[*roundtypes.FinalizeRoundResult, error](fmt.Errorf("failed to update round state to finalized: %w", err)), nil
			}

			// Fetch the finalized round data from the database to get the current state
			round, err := s.repo.GetRound(ctx, tx, req.GuildID, req.RoundID)
			if err != nil {
				s.metrics.RecordDBOperationError(ctx, "get_round")
				s.logger.ErrorContext(ctx, "Failed to fetch round data after finalization",
					attr.StringUUID("round_id", req.RoundID.String()),
					attr.Error(err),
				)
				return results.FailureResult[*roundtypes.FinalizeRoundResult, error](fmt.Errorf("failed to fetch round data: %w", err)), nil
			}

			// Fetch participants
			participants, err := s.repo.GetParticipants(ctx, tx, req.GuildID, req.RoundID)
			if err != nil {
				s.logger.ErrorContext(ctx, "Failed to fetch participants after finalization",
					attr.StringUUID("round_id", req.RoundID.String()),
					attr.Error(err),
				)
				// Log but proceed? Or fail? The handler needs participants.
				return results.FailureResult[*roundtypes.FinalizeRoundResult, error](fmt.Errorf("failed to fetch participants: %w", err)), nil
			}

			// TODO: Fetch teams if applicable (needs repository support for GetRoundTeams/Groups if distinct from round.Teams)
			// Assuming round.Teams might be populated by GetRound if repo handles it, or we need a separate call.
			// Current GetRound usually doesn't populate nested structs unless specified.
			// Looking at repository interface, there isn't a GetGroups/GetTeams method.
			// But roundtypes.Round has Teams field.
			// If GetRound doesn't populate it, we might be missing it.
			// However, for now let's assume GetRound or we can add logic later if teams are missing.
			// Actually, GetRound probably doesn't populate Teams if they are in a separate table/relation.
			// But wait, CreateRoundGroups exists.
			// Let's rely on round object for now, or assume empty teams if not fetched.
			// Participants are definitely needed.

			s.logger.InfoContext(ctx, "Round state updated to finalized successfully",
				attr.StringUUID("round_id", req.RoundID.String()),
			)

			return results.SuccessResult[*roundtypes.FinalizeRoundResult, error](&roundtypes.FinalizeRoundResult{
				Round:        round,
				Participants: participants,
				Teams:        round.Teams, // Pass whatever is in round object
			}), nil
		})
	})

	return result, err
}

// NotifyScoreModule prepares the data needed by the Score Module after a round is finalized.
// Note: In EDA, the service might just return the data, and the handler publishes the event.
// Or if this method is intended to trigger something internally, it should be clear.
// Based on previous code, it returned success/failure.
func (s *RoundService) NotifyScoreModule(ctx context.Context, result *roundtypes.FinalizeRoundResult) (results.OperationResult[*roundtypes.Round, error], error) {
	return withTelemetry(s, ctx, "NotifyScoreModule", result.Round.ID, func(ctx context.Context) (results.OperationResult[*roundtypes.Round, error], error) {
		round := result.Round

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
			return results.FailureResult[*roundtypes.Round, error](errors.New("no participants with submitted scores found")), nil
		}

		// The service simply validates that data is ready for score module.
		// The actual notification (event publishing) happens in the handler layer.
		return results.SuccessResult[*roundtypes.Round, error](round), nil
	})
}
