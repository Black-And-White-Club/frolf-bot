package scoreservice

import (
	"context"
	"errors"
	"fmt"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/uptrace/bun"
)

// ScoreOperationResult aligns with results.OperationResult[*sharedtypes.ScoreInfo, error].
type ScoreOperationResult = results.OperationResult[sharedtypes.ScoreInfo, error]

// CorrectScore orchestrates telemetry and transactions.
func (s *ScoreService) CorrectScore(
	ctx context.Context,
	guildID sharedtypes.GuildID,
	roundID sharedtypes.RoundID,
	userID sharedtypes.DiscordID,
	score sharedtypes.Score,
	tagNumber *sharedtypes.TagNumber,
) (ScoreOperationResult, error) {

	// Named transaction function for readability
	correctScoreTx := func(ctx context.Context, db bun.IDB) (ScoreOperationResult, error) {
		return s.executeCorrectScore(ctx, db, guildID, roundID, userID, score, tagNumber)
	}

	// Wrap with telemetry & transaction
	result, err := withTelemetry(s, ctx, "CorrectScore", roundID, func(ctx context.Context) (ScoreOperationResult, error) {
		return runInTx(s, ctx, correctScoreTx)
	})

	if err != nil {
		// This avoids "CorrectScore failed: CorrectScore failed: ..." nested messages.
		return ScoreOperationResult{}, err
	}

	// Return the domain result (Success or Failure payload)
	return result, nil
}

// executeCorrectScore contains the core business logic.
func (s *ScoreService) executeCorrectScore(
	ctx context.Context,
	db bun.IDB,
	guildID sharedtypes.GuildID,
	roundID sharedtypes.RoundID,
	userID sharedtypes.DiscordID,
	score sharedtypes.Score,
	tagNumber *sharedtypes.TagNumber,
) (ScoreOperationResult, error) {

	// 1. Domain Validation
	if score < -36 || score > 72 {
		// Wrap the sentinel error to provide context (the value) while keeping it comparable via errors.Is
		return results.FailureResult[sharedtypes.ScoreInfo, error](
			fmt.Errorf("%w: %d (must be between -36 and 72)", ErrInvalidScore, score),
		), nil
	}

	// 2. Logic: Handle tag number preservation
	effectiveTag := tagNumber
	if effectiveTag == nil {
		existingScores, err := s.repo.GetScoresForRound(ctx, db, guildID, roundID)
		if err != nil {
			// Infrastructure error: Return as error to trigger retry
			return ScoreOperationResult{}, fmt.Errorf("failed to fetch existing scores: %w", err)
		}

		for _, si := range existingScores {
			if si.UserID == userID && si.TagNumber != nil {
				tn := *si.TagNumber
				effectiveTag = &tn
				break
			}
		}
	}

	scoreInfo := &sharedtypes.ScoreInfo{
		UserID:    userID,
		Score:     score,
		TagNumber: effectiveTag,
	}

	// 3. Persistence
	if err := s.repo.UpdateOrAddScore(ctx, db, guildID, roundID, *scoreInfo); err != nil {
		// If the repo returns a domain error (e.g. Round Locked, Conflict), return FailureResult.
		// If it's a DB connection error, return 'err'.
		if errors.Is(err, ErrScoresAlreadyExist) {
			return results.FailureResult[sharedtypes.ScoreInfo, error](err), nil
		}

		// Unhandled infrastructure error
		return ScoreOperationResult{}, fmt.Errorf("failed to update score: %w", err)
	}

	return results.SuccessResult[sharedtypes.ScoreInfo, error](*scoreInfo), nil
}
