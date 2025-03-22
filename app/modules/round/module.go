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
	logger        observability.Logger
	metrics       observability.Metrics
	tracer        observability.Tracer
	config        *config.Config
	RoundRouter   *roundrouter.RoundRouter
	cancelFunc    context.CancelFunc
	helper        utils.Helpers
	observability observability.Observability // Still useful to have the complete object
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
	// Extract the components once during initialization
	logger := obs.GetLogger()
	metrics := obs.GetMetrics()
	tracer := obs.GetTracer()

	logger.Info("round.NewRoundModule called")

	// Initialize round service with observability components
	roundService := roundservice.NewRoundService(roundDB, eventBus, logger, metrics, tracer)

	// Initialize round router with observability
	roundRouter := roundrouter.NewRoundRouter(logger, router, eventBus, eventBus, cfg, helpers, tracer)

	// Configure the router with the round service.
	if err := roundRouter.Configure(roundService, eventBus); err != nil {
		return nil, fmt.Errorf("failed to configure round router: %w", err)
	}

	module := &Module{
		EventBus:      eventBus,
		RoundService:  roundService,
		logger:        logger,
		metrics:       metrics,
		tracer:        tracer,
		config:        cfg,
		RoundRouter:   roundRouter,
		helper:        helpers,
		observability: obs,
	}

	return module, nil
}

func (m *Module) Run(ctx context.Context, wg *sync.WaitGroup) {
	m.logger.Info("Starting round module")

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
	m.logger.Info("Round module goroutine stopped")
}

func (m *Module) Close() error {
	m.logger.Info("Stopping round module")

	// Cancel any other running operations
	if m.cancelFunc != nil {
		m.cancelFunc()
	}

	m.logger.Info("Round module stopped")
	return nil
}
