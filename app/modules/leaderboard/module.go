package leaderboard

import (
	"context"
	"log/slog"

	leaderboardservice "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/application"
	leaderboardhandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/infrastructure/handlers"
	leaderboarddb "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/infrastructure/repositories"
	leaderboardsubscribers "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/infrastructure/subscribers"
	"github.com/Black-And-White-Club/tcr-bot/app/shared"
	"github.com/Black-And-White-Club/tcr-bot/config"
)

// Module represents the leaderboard module.
type Module struct {
	EventBus         shared.EventBus
	Service          leaderboardservice.Service
	Handlers         leaderboardhandlers.Handlers
	Subscribers      leaderboardsubscribers.Subscribers
	logger           *slog.Logger
	config           *config.Config
	SubscribersReady chan struct{}
}

// NewLeaderboardModule creates a new LeaderboardModule with the provided dependencies.
func NewLeaderboardModule(ctx context.Context, cfg *config.Config, logger *slog.Logger, leaderboardDB leaderboarddb.LeaderboardDB, eventBus shared.EventBus) (*Module, error) {
	// Initialize leaderboard service and handlers
	leaderboardService := leaderboardservice.NewLeaderboardService(leaderboardDB, eventBus, logger)
	leaderboardHandlers := leaderboardhandlers.NewLeaderboardHandlers(leaderboardService, eventBus, logger)

	// Initialize leaderboard subscribers
	leaderboardSubscribers := leaderboardsubscribers.NewLeaderboardSubscribers(eventBus, leaderboardHandlers, logger)

	module := &Module{
		EventBus:         eventBus,
		Service:          leaderboardService,
		Handlers:         leaderboardHandlers,
		Subscribers:      leaderboardSubscribers,
		logger:           logger,
		config:           cfg,
		SubscribersReady: make(chan struct{}),
	}

	// Subscribe to leaderboard events in a separate goroutine
	go func() {
		subscriberCtx := context.Background() // Use a background context for subscribers

		if err := module.Subscribers.SubscribeToLeaderboardEvents(subscriberCtx); err != nil {
			logger.Error("Failed to subscribe to leaderboard events", "error", err)
			return
		}

		logger.Info("Leaderboard module subscribers are ready")
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
