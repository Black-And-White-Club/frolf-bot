package roundservice

import (
	"context"

	"log/slog"

	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/infrastructure/repositories"
	"github.com/Black-And-White-Club/tcr-bot/app/shared"
)

// RoundService handles round-related logic.
type RoundService struct {
	RoundDB  rounddb.RoundDB
	eventBus shared.EventBus
	logger   *slog.Logger
}

// NewRoundService creates a new RoundService.
func NewRoundService(ctx context.Context, eventBus shared.EventBus, db rounddb.RoundDB, logger *slog.Logger) Service {
	return &RoundService{
		RoundDB:  db,
		eventBus: eventBus,
		logger:   logger,
	}
}
