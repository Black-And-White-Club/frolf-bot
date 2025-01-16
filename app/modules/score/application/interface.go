package scoreservice

import (
	"context"

	scoreevents "github.com/Black-And-White-Club/tcr-bot/app/modules/score/domain/events"
)

// Service defines the interface for the ScoreService.
type Service interface {
	// Processes scores received from the round module and publishes leaderboard updates.
	ProcessRoundScores(ctx context.Context, event scoreevents.ScoresReceivedEvent) error

	// Corrects an individual score and triggers a leaderboard update.
	CorrectScore(ctx context.Context, event scoreevents.ScoreCorrectedEvent) error
}
