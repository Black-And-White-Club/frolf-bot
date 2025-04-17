package round

import (
	"context"
	"fmt"
	"sync"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
	roundrouter "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/router"
	"github.com/Black-And-White-Club/frolf-bot/config"
	"github.com/ThreeDotsLabs/watermill/message"
)

// Module represents the round module.
type Module struct {
	EventBus      eventbus.EventBus
	RoundService  roundservice.Service
	config        *config.Config
	RoundRouter   *roundrouter.RoundRouter
	cancelFunc    context.CancelFunc
	helper        utils.Helpers
	observability observability.Observability
}

// NewRoundModule creates a new instance of the Round module.
func NewRoundModule(
	ctx context.Context,
	cfg *config.Config,
	obs observability.Observability,
	roundDB rounddb.RoundDB,
	eventBus eventbus.EventBus,
	router *message.Router,
	helpers utils.Helpers,
) (*Module, error) {
	// Extract observability components
	logger := obs.Provider.Logger
	metrics := obs.Registry.RoundMetrics
	tracer := obs.Registry.Tracer

	logger.InfoContext(ctx, "round.NewRoundModule called")

	// Initialize round service with observability components
	roundService := roundservice.NewRoundService(roundDB, logger, metrics, tracer)

	// Initialize round router with observability
	roundRouter := roundrouter.NewRoundRouter(logger, router, eventBus, eventBus, cfg, helpers, tracer)

	// Configure the router with the round service.
	if err := roundRouter.Configure(roundService, eventBus, metrics); err != nil {
		return nil, fmt.Errorf("failed to configure round router: %w", err)
	}

	module := &Module{
		EventBus:      eventBus,
		RoundService:  roundService,
		config:        cfg,
		RoundRouter:   roundRouter,
		helper:        helpers,
		observability: obs,
	}

	return module, nil
}

// Run starts the round module.
func (m *Module) Run(ctx context.Context, wg *sync.WaitGroup) {
	logger := m.observability.Provider.Logger
	logger.InfoContext(ctx, "Starting round module")

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
	logger.InfoContext(ctx, "Round module goroutine stopped")
}

// Close stops the round module and cleans up resources.
func (m *Module) Close() error {
	logger := m.observability.Provider.Logger
	logger.Info("Stopping round module")

	// Cancel any other running operations
	if m.cancelFunc != nil {
		m.cancelFunc()
	}

	logger.Info("Round module stopped")
	return nil
}
