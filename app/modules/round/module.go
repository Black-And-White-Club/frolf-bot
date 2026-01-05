package round

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	roundadapters "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/adapters"
	roundqueue "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/queue"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
	roundrouter "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/router"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/utils"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/config"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/prometheus/client_golang/prometheus"
)

// Module represents the round module.
type Module struct {
	EventBus           eventbus.EventBus
	RoundService       roundservice.Service
	QueueService       roundqueue.QueueService
	config             *config.Config
	RoundRouter        *roundrouter.RoundRouter
	cancelFunc         context.CancelFunc
	helper             utils.Helpers
	observability      observability.Observability
	prometheusRegistry *prometheus.Registry
}

// NewRoundModule creates a new instance of the Round module.
func NewRoundModule(
	ctx context.Context,
	cfg *config.Config,
	obs observability.Observability,
	roundDB rounddb.RoundDB,
	userDB userdb.UserDB,
	eventBus eventbus.EventBus,
	router *message.Router,
	helpers utils.Helpers,
	routerCtx context.Context,
) (*Module, error) {
	logger := obs.Provider.Logger
	metrics := obs.Registry.RoundMetrics
	tracer := obs.Registry.Tracer

	logger.InfoContext(ctx, "round.NewRoundModule called")

	// Get the underlying Bun DB from your existing roundDB implementation
	roundDBImpl, ok := roundDB.(*rounddb.RoundDBImpl)
	if !ok {
		return nil, fmt.Errorf("roundDB is not of type *RoundDBImpl")
	}

	// Initialize River queue service using the existing Bun DB
	queueService, err := roundqueue.NewService(
		ctx,
		roundDBImpl.DB, // Use the existing Bun DB connection
		logger,
		cfg.Postgres.DSN,
		metrics,
		eventBus,
		helpers,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize queue service: %w", err)
	}

	// Start the queue service
	if err := queueService.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start queue service: %w", err)
	}

	// Use your existing round validator
	roundValidator := roundutil.NewRoundValidator()

	// Initialize round service with queue service and score DB
	roundService := roundservice.NewRoundService(
		roundDB,
		queueService,
		eventBus,
		roundadapters.NewUserLookupAdapter(userDB),
		metrics,
		logger,
		tracer,
		roundValidator,
	)

	// Create a Prometheus registry for this module
	prometheusRegistry := prometheus.NewRegistry()

	// Initialize round router
	roundRouter := roundrouter.NewRoundRouter(logger, router, eventBus, eventBus, cfg, helpers, tracer, prometheusRegistry)

	// Configure the router with the round service
	if err := roundRouter.Configure(routerCtx, roundService, eventBus, metrics); err != nil {
		return nil, fmt.Errorf("failed to configure round router: %w", err)
	}

	module := &Module{
		EventBus:           eventBus,
		RoundService:       roundService,
		QueueService:       queueService,
		config:             cfg,
		RoundRouter:        roundRouter,
		helper:             helpers,
		observability:      obs,
		prometheusRegistry: prometheusRegistry,
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

	// Stop the queue service
	if m.QueueService != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := m.QueueService.Stop(ctx); err != nil {
			logger.Error("Error stopping queue service", "error", err)
		}
	}

	// Close the RoundRouter
	if m.RoundRouter != nil {
		if err := m.RoundRouter.Close(); err != nil {
			logger.Error("Error closing RoundRouter from module", "error", err)
			return fmt.Errorf("error closing RoundRouter: %w", err)
		}
	}

	logger.Info("Round module stopped")
	return nil
}
