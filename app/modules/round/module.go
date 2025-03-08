package round

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/Black-And-White-Club/frolf-bot-shared/errors"
	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
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
	logger        *slog.Logger
	config        *config.Config
	RoundRouter   *roundrouter.RoundRouter
	cancelFunc    context.CancelFunc
	ErrorReporter errors.ErrorReporterInterface
	helper        utils.Helpers
}

// NewRoundModule creates a new instance of the Round module.
func NewRoundModule(ctx context.Context, cfg *config.Config, logger *slog.Logger, roundDB rounddb.RoundDBInterface, eventBus eventbus.EventBus, router *message.Router, ErrorReporter errors.ErrorReporterInterface, helper utils.Helpers) (*Module, error) {
	logger.Info("round.NewRoundModule called")

	// Initialize round service.
	roundService := roundservice.NewRoundService(roundDB, eventBus, logger, ErrorReporter)

	// Initialize round router.
	roundRouter := roundrouter.NewRoundRouter(logger, router, eventBus, helper)

	// Configure the router with the round service.
	if err := roundRouter.Configure(roundService); err != nil {
		return nil, fmt.Errorf("failed to configure round router: %w", err)
	}

	module := &Module{
		EventBus:     eventBus,
		RoundService: roundService,
		logger:       logger,
		config:       cfg,
		RoundRouter:  roundRouter, // Set the RoundRouter
	}

	return module, nil
}

func (m *Module) Run(ctx context.Context, wg *sync.WaitGroup) {
	m.logger.Info("Starting round module")

	// Create a context that can be canceled
	ctx, cancel := context.WithCancel(ctx)
	m.cancelFunc = cancel
	defer cancel()

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
