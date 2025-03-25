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
)

// Module represents the score module.
type Module struct {
	EventBus      eventbus.EventBus
	ScoreService  scoreservice.Service
	logger        observability.Logger
	metrics       observability.Metrics
	tracer        observability.Tracer
	config        *config.Config
	ScoreRouter   *scorerouter.ScoreRouter
	cancelFunc    context.CancelFunc
	helper        utils.Helpers
	observability observability.Observability // Still useful to have the complete object
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
) (*Module, error) {
	// Extract the components once during initialization
	logger := obs.GetLogger()
	metrics := obs.GetMetrics()
	tracer := obs.GetTracer()

	logger.Info("score.NewScoreModule called")

	// Initialize score service with observability components
	scoreService := scoreservice.NewScoreService(eventBus, scoreDB, logger, metrics, tracer)

	// Initialize score router with observability
	scoreRouter := scorerouter.NewScoreRouter(logger, router, eventBus, eventBus, cfg, helpers, tracer)

	// Configure the router with the score service.
	if err := scoreRouter.Configure(scoreService, eventBus); err != nil {
		return nil, fmt.Errorf("failed to configure score router: %w", err)
	}

	module := &Module{
		EventBus:      eventBus,
		ScoreService:  scoreService,
		logger:        logger,
		metrics:       metrics,
		tracer:        tracer,
		config:        cfg,
		ScoreRouter:   scoreRouter,
		helper:        helpers,
		observability: obs,
	}

	return module, nil
}

func (m *Module) Run(ctx context.Context, wg *sync.WaitGroup) {
	m.logger.Info("Starting score module")

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
	m.logger.Info("Score module goroutine stopped")
}

func (m *Module) Close() error {
	m.logger.Info("Stopping score module")

	// Cancel any other running operations
	if m.cancelFunc != nil {
		m.cancelFunc()
	}

	m.logger.Info("Score module stopped")
	return nil
}
