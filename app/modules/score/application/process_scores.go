package scoreservice

import (
	"context"
	"fmt"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/uptrace/bun"
)

type ProcessRoundScoresResult struct{ TagMappings []sharedtypes.TagMapping }

// 1. Update the signature to use the specific Result type instead of the generic ScoreOperationResult alias
func (s *ScoreService) ProcessRoundScores(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	roundID sharedtypes.RoundID,
	scores []sharedtypes.ScoreInfo,
	overwrite bool,
) (results.OperationResult[ProcessRoundScoresResult, error], error) {

	// Update the internal closure return type as well
	processScoresTx := func(ctx context.Context, db bun.IDB) (results.OperationResult[ProcessRoundScoresResult, error], error) {
		return s.executeProcessRoundScores(ctx, db, guildID, roundID, scores, overwrite)
	}

	result, err := withTelemetry(s, ctx, "ProcessRoundScores", roundID, func(ctx context.Context) (results.OperationResult[ProcessRoundScoresResult, error], error) {
		return runInTx(s, ctx, processScoresTx)
	})

	if err != nil {
		return results.OperationResult[ProcessRoundScoresResult, error]{}, err
	}

	return result, nil
}

func (s *ScoreService) executeProcessRoundScores(
	ctx context.Context,
	db bun.IDB,
	guildID sharedtypes.GuildID,
	roundID sharedtypes.RoundID,
	scores []sharedtypes.ScoreInfo,
	overwrite bool,
) (results.OperationResult[ProcessRoundScoresResult, error], error) {

	// 1. Check for existing scores
	existingScores, err := s.repo.GetScoresForRound(ctx, db, guildID, roundID)
	if err != nil {
		// Hard error (infrastructure)
		return results.OperationResult[ProcessRoundScoresResult, error]{}, fmt.Errorf("failed to check existing scores: %w", err)
	}

	if len(existingScores) > 0 && !overwrite {
		// Business Failure - Note the explicit type parameters
		return results.FailureResult[ProcessRoundScoresResult, error](ErrScoresAlreadyExist), nil
	}

	// 2. Logic: Process scores
	processedScores, err := s.ProcessScoresForStorage(ctx, guildID, roundID, scores)
	if err != nil {
		return results.FailureResult[ProcessRoundScoresResult, error](
			fmt.Errorf("score processing failed: %w", err),
		), nil
	}

	// 3. Persistence
	if err := s.repo.LogScores(ctx, db, guildID, roundID, processedScores, "auto"); err != nil {
		return results.OperationResult[ProcessRoundScoresResult, error]{}, fmt.Errorf("failed to log scores: %w", err)
	}

	// 4. Mapping (The payoff!)
	tagMappings := make([]sharedtypes.TagMapping, 0, len(processedScores))
	for _, scoreInfo := range processedScores {
		if scoreInfo.TagNumber != nil {
			tagMappings = append(tagMappings, sharedtypes.TagMapping{
				DiscordID: scoreInfo.UserID,
				TagNumber: *scoreInfo.TagNumber,
			})
			s.metrics.RecordPlayerTag(ctx, roundID, scoreInfo.UserID, scoreInfo.TagNumber)
		}
	}

	s.metrics.RecordScoreProcessingSuccess(ctx, roundID)

	// Return the actual data the handler is waiting for
	return results.SuccessResult[ProcessRoundScoresResult, error](ProcessRoundScoresResult{
		TagMappings: tagMappings,
	}), nil
}
