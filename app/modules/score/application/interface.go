package scoreservice

import (
	"context"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// Service is the interface for score service, as you provided
type Service interface {
	// Processes scores received from the round module and publishes leaderboard updates.
	ProcessRoundScores(ctx context.Context, roundID sharedtypes.RoundID, scores []sharedtypes.ScoreInfo) (ScoreOperationResult, error)

	// Corrects an individual score and triggers a leaderboard update.
	CorrectScore(ctx context.Context, roundID sharedtypes.RoundID, userID sharedtypes.DiscordID, score sharedtypes.Score, tagNumber *sharedtypes.TagNumber) (ScoreOperationResult, error)
	ProcessScoresForStorage(ctx context.Context, roundID sharedtypes.RoundID, scores []sharedtypes.ScoreInfo) ([]sharedtypes.ScoreInfo, error)
}
