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

	roundDBImpl, ok := roundDB.(*rounddb.RoundDBImpl)
	if !ok {
		return nil, fmt.Errorf("roundDB is not of type *RoundDBImpl")
	}

	queueService, err := roundqueue.NewService(
		ctx,
		roundDBImpl.DB,
		logger,
		cfg.Postgres.DSN,
		metrics,
		eventBus,
		helpers,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize queue service: %w", err)
	}

	if err := queueService.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start queue service: %w", err)
	}

	roundValidator := roundutil.NewRoundValidator()

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

	prometheusRegistry := prometheus.NewRegistry()

	// Initialize round router
	roundRouter := roundrouter.NewRoundRouter(logger, router, eventBus, eventBus, helpers, tracer, prometheusRegistry)

	// CRITICAL FIX: Configure registers the handlers and adds middleware (CorrelationID)
	// Without this, the router is created but never listens to any topics.
	if err := roundRouter.Configure(roundService, metrics); err != nil {
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

func (m *Module) Run(ctx context.Context, wg *sync.WaitGroup) {
	logger := m.observability.Provider.Logger
	logger.InfoContext(ctx, "Starting round module")

	ctx, cancel := context.WithCancel(ctx)
	m.cancelFunc = cancel
	defer cancel()

	if wg != nil {
		defer wg.Done()
	}

	<-ctx.Done()
	logger.InfoContext(ctx, "Round module goroutine stopped")
}

func (m *Module) Close() error {
	logger := m.observability.Provider.Logger
	logger.Info("Stopping round module")

	if m.cancelFunc != nil {
		m.cancelFunc()
	}

	if m.QueueService != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := m.QueueService.Stop(ctx); err != nil {
			logger.Error("Error stopping queue service", "error", err)
		}
	}

	if m.RoundRouter != nil {
		if err := m.RoundRouter.Close(); err != nil {
			logger.Error("Error closing RoundRouter from module", "error", err)
			return fmt.Errorf("error closing RoundRouter: %w", err)
		}
	}

	logger.Info("Round module stopped")
	return nil
}
