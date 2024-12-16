package scorerouter

import (
	"context"
	"log"

	scorecommands "github.com/Black-And-White-Club/tcr-bot/app/modules/score/commands"
	scoredb "github.com/Black-And-White-Club/tcr-bot/app/modules/score/db"
	"github.com/Black-And-White-Club/tcr-bot/internal/commands"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
)

// ScoreCommandBus is the command bus for the score module.
type ScoreCommandBus struct {
	publisher message.Publisher
	marshaler cqrs.CommandEventMarshaler
}

// NewScoreCommandBus creates a new ScoreCommandBus.
func NewScoreCommandBus(publisher message.Publisher, marshaler cqrs.CommandEventMarshaler) *ScoreCommandBus {
	return &ScoreCommandBus{publisher: publisher, marshaler: marshaler}
}

func (r ScoreCommandBus) Send(ctx context.Context, cmd commands.Command) error {
	return watermillutil.SendCommand(ctx, r.publisher, r.marshaler, cmd, cmd.CommandName())
}

// ScoreCommandRouter implements the CommandRouter interface.
type ScoreCommandRouter struct {
	commandBus CommandBus
}

// NewScoreCommandRouter creates a new ScoreCommandRouter.
func NewScoreCommandRouter(commandBus CommandBus) CommandRouter {
	return &ScoreCommandRouter{commandBus: commandBus}
}

// UpdateScores handles score update logic.
func (s *ScoreCommandRouter) UpdateScores(ctx context.Context, roundID string, scores []scoredb.Score) error {
	updateScoresCmd := scorecommands.UpdateScoresCommand{
		RoundID: roundID,
		Scores:  scores,
	}

	log.Printf("Sending UpdateScoresCommand: %+v\n", updateScoresCmd)

	return s.commandBus.Send(ctx, updateScoresCmd)
}
