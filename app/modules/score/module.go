package score

import (
	"context"
	"log/slog"

	scoreservice "github.com/Black-And-White-Club/tcr-bot/app/modules/score/application"
	scorehandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/score/infrastructure/handlers"
	scoredb "github.com/Black-And-White-Club/tcr-bot/app/modules/score/infrastructure/repositories"
	scoresubscribers "github.com/Black-And-White-Club/tcr-bot/app/modules/score/infrastructure/subscribers"
	"github.com/Black-And-White-Club/tcr-bot/app/shared"
	"github.com/Black-And-White-Club/tcr-bot/config"
)

// Module represents the score module.
type Module struct {
	EventBus         shared.EventBus
	Service          scoreservice.Service
	Handlers         scorehandlers.Handlers
	Subscribers      scoresubscribers.Subscribers
	logger           *slog.Logger
	config           *config.Config
	SubscribersReady chan struct{} // Channel to signal subscriber readiness
}

func NewScoreModule(ctx context.Context, cfg *config.Config, logger *slog.Logger, scoreDB scoredb.ScoreDB, eventBus shared.EventBus) (*Module, error) {
	scoreService := scoreservice.NewScoreService(eventBus, scoreDB, logger)
	scoreHandlers := scorehandlers.NewScoreHandlers(scoreService, eventBus, logger)
	scoreSubscribers := scoresubscribers.NewSubscribers(eventBus, scoreHandlers, logger)

	module := &Module{
		EventBus:         eventBus,
		Service:          scoreService,
		Handlers:         scoreHandlers,
		Subscribers:      scoreSubscribers,
		logger:           logger,
		config:           cfg,
		SubscribersReady: make(chan struct{}), // Initialize the channel
	}

	// Start the subscription process in a separate goroutine
	go func() {
		subscriberCtx := context.Background()

		if err := module.Subscribers.SubscribeToScoreEvents(subscriberCtx); err != nil {
			logger.Error("Failed to subscribe to score events", slog.Any("error", err))
			return
		}

		logger.Info("Score module subscribers are ready")

		// Signal that subscribers are ready
		close(module.SubscribersReady)
	}()

	return module, nil
}

// IsInitialized safely checks module initialization
func (m *Module) IsInitialized() bool {
	select {
	case <-m.SubscribersReady:
		return true
	default:
		return false
	}
}
