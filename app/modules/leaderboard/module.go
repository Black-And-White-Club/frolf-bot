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
)

// Module represents the leaderboard module.
type Module struct {
	EventBus           eventbus.EventBus
	LeaderboardService leaderboardservice.Service
	logger             observability.Logger
	metrics            observability.Metrics
	tracer             observability.Tracer
	config             *config.Config
	LeaderboardRouter  *leaderboardrouter.LeaderboardRouter
	cancelFunc         context.CancelFunc
	helper             utils.Helpers
	observability      observability.Observability
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
) (*Module, error) {
	// Extract observability components
	logger := obs.GetLogger()
	metrics := obs.GetMetrics()
	tracer := obs.GetTracer()

	logger.Info("leaderboard.NewLeaderboardModule called")

	// Initialize leaderboard service with observability components
	leaderboardService := leaderboardservice.NewLeaderboardService(leaderboardDB, eventBus, logger, metrics, tracer)

	// Initialize leaderboard router with observability
	leaderboardRouter := leaderboardrouter.NewLeaderboardRouter(logger, router, eventBus, eventBus, cfg, helpers, tracer)

	// Configure the router with the leaderboard service
	if err := leaderboardRouter.Configure(leaderboardService, eventBus); err != nil {
		return nil, fmt.Errorf("failed to configure leaderboard router: %w", err)
	}

	module := &Module{
		EventBus:           eventBus,
		LeaderboardService: leaderboardService,
		logger:             logger,
		metrics:            metrics,
		tracer:             tracer,
		config:             cfg,
		LeaderboardRouter:  leaderboardRouter,
		helper:             helpers,
		observability:      obs,
	}

	return module, nil
}

// Run starts the leaderboard module.
func (m *Module) Run(ctx context.Context, wg *sync.WaitGroup) {
	m.logger.Info("Starting leaderboard module")

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
	m.logger.Info("Leaderboard module goroutine stopped")
}

// Close stops the leaderboard module and cleans up resources.
func (m *Module) Close() error {
	m.logger.Info("Stopping leaderboard module")

	// Cancel any other running operations
	if m.cancelFunc != nil {
		m.cancelFunc()
	}

	m.logger.Info("Leaderboard module stopped")
	return nil
}
