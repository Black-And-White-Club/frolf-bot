package round

import (
	"context"
	"log/slog"

	roundservice "github.com/Black-And-White-Club/tcr-bot/app/modules/round/application"
	roundhandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/round/infrastructure/handlers"
	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/infrastructure/repositories"
	roundsubscribers "github.com/Black-And-White-Club/tcr-bot/app/modules/round/infrastructure/subscribers"
	"github.com/Black-And-White-Club/tcr-bot/app/shared"
	"github.com/Black-And-White-Club/tcr-bot/config"
)

// Module represents the round module.
type Module struct {
	EventBus         shared.EventBus
	RoundService     roundservice.Service
	Handlers         roundhandlers.Handlers
	Subscribers      roundsubscribers.Subscribers
	logger           *slog.Logger
	config           *config.Config
	SubscribersReady chan struct{}
}

// NewRoundModule creates a new instance of the Round module.
func NewRoundModule(ctx context.Context, cfg *config.Config, logger *slog.Logger, roundDB rounddb.RoundDB, eventBus shared.EventBus) (*Module, error) {
	logger.Info("round.NewRoundModule called")

	// Initialize round service.
	roundService := roundservice.NewRoundService(ctx, eventBus, roundDB, logger)

	// Initialize round handlers.
	roundHandlers := roundhandlers.NewRoundHandlers(roundService, &eventBus, logger)

	// Initialize round subscribers.
	roundSubscribers := roundsubscribers.NewRoundSubscribers(eventBus, roundHandlers, logger) // Pass the handler directly

	module := &Module{
		EventBus:         eventBus,
		RoundService:     roundService,
		Handlers:         roundHandlers,
		Subscribers:      roundSubscribers,
		logger:           logger,
		config:           cfg,
		SubscribersReady: make(chan struct{}),
	}

	// Start the subscription process in a separate goroutine.
	go func() {
		subscriberCtx := context.Background()

		if err := module.Subscribers.SubscribeToRoundManagementEvents(subscriberCtx); err != nil {
			logger.Error("Failed to subscribe to round management events", slog.Any("error", err))
			return
		}
		if err := module.Subscribers.SubscribeToParticipantManagementEvents(subscriberCtx); err != nil {
			logger.Error("Failed to subscribe to participant management events", slog.Any("error", err))
			return
		}
		if err := module.Subscribers.SubscribeToRoundFinalizationEvents(subscriberCtx); err != nil {
			logger.Error("Failed to subscribe to round finalization events", slog.Any("error", err))
			return
		}
		if err := module.Subscribers.SubscribeToRoundStartedEvents(subscriberCtx); err != nil {
			logger.Error("Failed to subscribe to round started events", slog.Any("error", err))
			return
		}

		logger.Info("Round module subscribers are ready")

		// Signal that initialization is complete
		close(module.SubscribersReady)
	}()

	return module, nil
}

// IsInitialized checks if the subscribers are ready.
func (m *Module) IsInitialized() bool {
	select {
	case <-m.SubscribersReady:
		return true
	default:
		return false
	}
}
