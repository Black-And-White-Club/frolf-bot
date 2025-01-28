package roundhandlers

import (
	"log/slog"

	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
)

// RoundHandlers handles round-related events.
type RoundHandlers struct {
	RoundService roundservice.Service
	logger       *slog.Logger
}

// NewRoundHandlers creates a new RoundHandlers.
func NewRoundHandlers(roundService roundservice.Service, logger *slog.Logger) Handlers {
	return &RoundHandlers{
		RoundService: roundService,
		logger:       logger,
	}
}
