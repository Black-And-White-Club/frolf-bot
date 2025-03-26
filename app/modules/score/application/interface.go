package scoreservice

import (
	"context"

	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	"github.com/ThreeDotsLabs/watermill/message"
)

// Service is the interface for score service, as you provided
type Service interface {
	// Processes scores received from the round module and publishes leaderboard updates.
	ProcessRoundScores(ctx context.Context, msg *message.Message, event scoreevents.ProcessRoundScoresRequestPayload) (ScoreOperationResult, error)

	// Corrects an individual score and triggers a leaderboard update.
	CorrectScore(ctx context.Context, msg *message.Message, event scoreevents.ScoreUpdateRequestPayload) (ScoreOperationResult, error)
}
