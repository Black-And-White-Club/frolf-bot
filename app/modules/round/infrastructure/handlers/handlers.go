package roundhandlers

import (
	"log/slog"

	roundservice "github.com/Black-And-White-Club/tcr-bot/app/modules/round/application"
	"github.com/Black-And-White-Club/tcr-bot/app/shared"
)

// RoundHandlers handles round-related events.
type RoundHandlers struct {
	RoundService roundservice.Service
	EventBus     *shared.EventBus
	logger       *slog.Logger
}

// NewRoundHandlers creates a new RoundHandlers.
func NewRoundHandlers(roundService roundservice.Service, eventBus *shared.EventBus, logger *slog.Logger) *RoundHandlers {
	return &RoundHandlers{
		RoundService: roundService,
		EventBus:     eventBus,
		logger:       logger,
	}
}
