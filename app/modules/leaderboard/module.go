package leaderboard

import (
	"context"
	"fmt"
	"sync"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	leaderboardrouter "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/router"
	"github.com/Black-And-White-Club/frolf-bot/config"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/prometheus/client_golang/prometheus"
)

// Module represents the leaderboard module.
type Module struct {
	EventBus           eventbus.EventBus
	LeaderboardService leaderboardservice.Service
	config             *config.Config
	LeaderboardRouter  *leaderboardrouter.LeaderboardRouter
	cancelFunc         context.CancelFunc
	Helper             utils.Helpers
	observability      observability.Observability
	prometheusRegistry *prometheus.Registry
}

// NewLeaderboardModule creates a new instance of the Leaderboard module.
func NewLeaderboardModule(
	ctx context.Context,
	cfg *config.Config,
	obs observability.Observability,
	leaderboardDB leaderboarddb.LeaderboardDB,
	eventBus eventbus.EventBus,
	router *message.Router,
	helpers utils.Helpers,
	routerCtx context.Context,
) (*Module, error) {
	// Extract observability components
	logger := obs.Provider.Logger
	metrics := obs.Registry.LeaderboardMetrics
	tracer := obs.Registry.Tracer

	logger.InfoContext(ctx, "leaderboard.NewLeaderboardModule called")

	// Initialize leaderboard service with observability components
	leaderboardService := leaderboardservice.NewLeaderboardService(leaderboardDB, eventBus, logger, metrics, tracer)

	// Create a Prometheus registry for this module (similar to score module)
	prometheusRegistry := prometheus.NewRegistry()

	// Initialize leaderboard router with observability and prometheusRegistry
	leaderboardRouter := leaderboardrouter.NewLeaderboardRouter(logger, router, eventBus, eventBus, cfg, helpers, tracer, prometheusRegistry)

	// Configure the router with the leaderboard service, passing routerCtx
	if err := leaderboardRouter.Configure(routerCtx, leaderboardService, eventBus, metrics); err != nil {
		return nil, fmt.Errorf("failed to configure leaderboard router: %w", err)
	}

	module := &Module{
		EventBus:           eventBus,
		LeaderboardService: leaderboardService,
		config:             cfg,
		LeaderboardRouter:  leaderboardRouter,
		Helper:             helpers,
		observability:      obs,
		prometheusRegistry: prometheusRegistry, // Assigned prometheusRegistry
	}

	return module, nil
}

// Run starts the leaderboard module.
func (m *Module) Run(ctx context.Context, wg *sync.WaitGroup) {
	logger := m.observability.Provider.Logger
	logger.InfoContext(ctx, "Starting leaderboard module")

	// Create a context that can be canceled
	ctx, cancel := context.WithCancel(ctx)
	m.cancelFunc = cancel
	defer cancel()

	// If we have a wait group, mark as done when this method exits
	if wg != nil {
		defer wg.Done()
	}

	// Keep this goroutine alive until the context is canceled
	<-ctx.Done()
	logger.InfoContext(ctx, "Leaderboard module goroutine stopped")
}

// Close stops the leaderboard module and cleans up resources.
func (m *Module) Close() error {
	logger := m.observability.Provider.Logger
	logger.Info("Stopping leaderboard module")

	// Cancel any other running operations
	if m.cancelFunc != nil {
		m.cancelFunc()
	}

	// Close the leaderboard router (similar to score module)
	if m.LeaderboardRouter != nil {
		if err := m.LeaderboardRouter.Close(); err != nil {
			logger.Error("Error closing LeaderboardRouter from module", "error", err)
			return fmt.Errorf("error closing LeaderboardRouter: %w", err)
		}
	}

	logger.Info("Leaderboard module stopped")
	return nil
}
