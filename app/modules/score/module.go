package score

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	scoreservice "github.com/Black-And-White-Club/frolf-bot/app/modules/score/application"
	scoredb "github.com/Black-And-White-Club/frolf-bot/app/modules/score/infrastructure/repositories"
	scorerouter "github.com/Black-And-White-Club/frolf-bot/app/modules/score/infrastructure/router"
	"github.com/Black-And-White-Club/frolf-bot/config"
	"github.com/ThreeDotsLabs/watermill/message"
)

// Module represents the score module.
type Module struct {
	EventBus     eventbus.EventBus
	ScoreService scoreservice.Service
	logger       *slog.Logger
	config       *config.Config
	ScoreRouter  *scorerouter.ScoreRouter
	cancelFunc   context.CancelFunc
}

func NewScoreModule(ctx context.Context, cfg *config.Config, logger *slog.Logger, scoreDB scoredb.ScoreDB, eventBus eventbus.EventBus, router *message.Router) (*Module, error) {
	logger.Info("score.NewScoreModule called")

	// Initialize score service.
	scoreService := scoreservice.NewScoreService(eventBus, scoreDB, logger)

	// Initialize score router.
	scoreRouter := scorerouter.NewScoreRouter(logger, router, eventBus)

	// Configure the router with the score service.
	if err := scoreRouter.Configure(scoreService); err != nil {
		return nil, fmt.Errorf("failed to configure score router: %w", err)
	}

	module := &Module{
		EventBus:     eventBus,
		ScoreService: scoreService,
		logger:       logger,
		config:       cfg,
		ScoreRouter:  scoreRouter, // Set the ScoreRouter
	}

	return module, nil
}

func (m *Module) Run(ctx context.Context, wg *sync.WaitGroup) {
	m.logger.Info("Starting score module")

	// Create a context that can be canceled
	ctx, cancel := context.WithCancel(ctx)
	m.cancelFunc = cancel
	defer cancel()

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
