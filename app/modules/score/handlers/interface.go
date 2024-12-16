package scorehandlers

import (
	"context"

	"github.com/Black-And-White-Club/tcr-bot/internal/commands"
)

// ScoreCommandHandler is an interface for handlers that process score commands.
type ScoreCommandHandler interface {
	Handle(ctx context.Context, cmd commands.Command) error
}
