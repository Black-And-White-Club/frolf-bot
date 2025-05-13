package score

import (
	"context"
	"fmt"
	"sync"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	scoreservice "github.com/Black-And-White-Club/frolf-bot/app/modules/score/application"
	scoredb "github.com/Black-And-White-Club/frolf-bot/app/modules/score/infrastructure/repositories"
	scorerouter "github.com/Black-And-White-Club/frolf-bot/app/modules/score/infrastructure/router"
	"github.com/Black-And-White-Club/frolf-bot/config"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/prometheus/client_golang/prometheus"
)

type Module struct {
	EventBus           eventbus.EventBus
	ScoreService       scoreservice.Service
	config             *config.Config
	ScoreRouter        *scorerouter.ScoreRouter
	cancelFunc         context.CancelFunc
	Helper             utils.Helpers
	observability      observability.Observability
	prometheusRegistry *prometheus.Registry
}

// NewScoreModule creates a new instance of the Score module.
func NewScoreModule(
	ctx context.Context,
	cfg *config.Config,
	obs observability.Observability,
	scoreDB scoredb.ScoreDB,
	eventBus eventbus.EventBus,
	router *message.Router,
	helpers utils.Helpers,
	routerCtx context.Context,
) (*Module, error) {
	logger := obs.Provider.Logger
	metrics := obs.Registry.ScoreMetrics
	tracer := obs.Registry.Tracer

	scoreService := scoreservice.NewScoreService(scoreDB, eventBus, logger, metrics, tracer)

	// Prometheus registry can be created here or passed in if it's a shared registry
	prometheusRegistry := prometheus.NewRegistry()

	// Create the ScoreRouter instance, passing the externally created router
	scoreRouter := scorerouter.NewScoreRouter(logger, router, eventBus, eventBus, cfg, helpers, tracer, prometheusRegistry)

	// Configure the ScoreRouter
	if err := scoreRouter.Configure(routerCtx, scoreService, eventBus, metrics); err != nil {
		return nil, fmt.Errorf("failed to configure score router: %w", err)
	}

	module := &Module{
		EventBus:           eventBus,
		ScoreService:       scoreService,
		config:             cfg,
		ScoreRouter:        scoreRouter,
		Helper:             helpers,
		observability:      obs,
		prometheusRegistry: prometheusRegistry,
	}

	return module, nil
}

func (m *Module) Run(ctx context.Context, wg *sync.WaitGroup) {
	ctx, cancel := context.WithCancel(ctx)
	m.cancelFunc = cancel
	defer cancel()

	if wg != nil {
		defer wg.Done()
	}

	<-ctx.Done()
}

func (m *Module) Close() error {
	logger := m.observability.Provider.Logger

	// Closing the module should trigger the ScoreRouter's close
	if m.ScoreRouter != nil {
		if err := m.ScoreRouter.Close(); err != nil {
			logger.Error("Error closing ScoreRouter from module", "error", err)
			return fmt.Errorf("error closing ScoreRouter: %w", err)
		}
	}

	if m.cancelFunc != nil {
		m.cancelFunc()
	}

	return nil
}
