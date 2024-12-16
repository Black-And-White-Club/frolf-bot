package scorerouter

import (
	"context"

	scoredb "github.com/Black-And-White-Club/tcr-bot/app/modules/score/db"
	"github.com/Black-And-White-Club/tcr-bot/internal/commands"
)

// CommandRouter is the interface for the command router
type CommandRouter interface {
	UpdateScores(ctx context.Context, roundID string, scores []scoredb.Score) error
}

// CommandBus is the interface for the command bus
type CommandBus interface {
	Send(ctx context.Context, cmd commands.Command) error
}
