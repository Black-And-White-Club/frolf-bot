package scoreservice

import (
	"context"

	scoreevents "github.com/Black-And-White-Club/tcr-bot/app/modules/score/events"
)

// ScoreService handles score processing logic.
type Service interface {
	ProcessRoundScores(ctx context.Context, event scoreevents.ScoresReceivedEvent) error
	CorrectScore(ctx context.Context, event scoreevents.ScoreCorrectedEvent) error
}
