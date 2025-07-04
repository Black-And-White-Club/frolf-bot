package scoredb

import (
	"context"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// ScoreDB is an interface for interacting with the score database.
type ScoreDB interface {
	LogScores(ctx context.Context, roundID sharedtypes.RoundID, scores []sharedtypes.ScoreInfo, source string) error
	UpdateScore(ctx context.Context, roundID sharedtypes.RoundID, userID sharedtypes.DiscordID, newScore sharedtypes.Score) error
	UpdateOrAddScore(ctx context.Context, roundID sharedtypes.RoundID, scoreInfo sharedtypes.ScoreInfo) error
	GetScoresForRound(ctx context.Context, roundID sharedtypes.RoundID) ([]sharedtypes.ScoreInfo, error)
}
