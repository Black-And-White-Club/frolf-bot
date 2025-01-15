package roundsubscribers

import (
	"log/slog"

	roundhandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/round/infrastructure/handlers"
	"github.com/Black-And-White-Club/tcr-bot/app/shared"
)

// RoundSubscribers subscribes to round-related events.
type RoundEventSubscribers struct {
	eventBus shared.EventBus
	logger   *slog.Logger
	handlers roundhandlers.Handlers
}

// NewRoundSubscribers creates a new RoundSubscribers.
func NewRoundSubscribers(eventBus shared.EventBus, handlers roundhandlers.Handlers, logger *slog.Logger) *RoundEventSubscribers {
	return &RoundEventSubscribers{
		eventBus: eventBus,
		logger:   logger,
		handlers: handlers,
	}
}
