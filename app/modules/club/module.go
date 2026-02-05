package club

import (
	"context"
	"fmt"
	"sync"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	clubmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/club"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	clubservice "github.com/Black-And-White-Club/frolf-bot/app/modules/club/application"
	clubhandlers "github.com/Black-And-White-Club/frolf-bot/app/modules/club/infrastructure/handlers"
	clubdb "github.com/Black-And-White-Club/frolf-bot/app/modules/club/infrastructure/repositories"
	clubrouter "github.com/Black-And-White-Club/frolf-bot/app/modules/club/infrastructure/router"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/uptrace/bun"
)

// Module represents the club module.
type Module struct {
	ClubService   clubservice.Service
	ClubRouter    *clubrouter.ClubRouter
	cancelFunc    context.CancelFunc
	observability observability.Observability
}

// NewClubModule creates and initializes a new club module.
func NewClubModule(
	ctx context.Context,
	obs observability.Observability,
	eventBus eventbus.EventBus,
	router *message.Router,
	helpers utils.Helpers,
	routerCtx context.Context,
	db *bun.DB,
) (*Module, error) {
	logger := obs.Provider.Logger
	tracer := obs.Registry.Tracer

	logger.InfoContext(ctx, "club.NewClubModule initializing")

	// 1. Initialize Repository
	repo := clubdb.NewClubRepository()

	// 2. Initialize Metrics (noop for now, real metrics wired via observability registry)
	metrics := clubmetrics.NewNoop()

	// 3. Initialize Service
	service := clubservice.NewClubService(repo, logger, metrics, tracer, db)

	// 3. Initialize Handlers
	handlers := clubhandlers.NewClubHandlers(service, logger, tracer)

	// 4. Initialize Router
	clubRouter := clubrouter.NewClubRouter(
		logger,
		router,
		eventBus,
		eventBus,
		helpers,
		tracer,
	)

	// 5. Configure the router with handlers
	if err := clubRouter.Configure(routerCtx, handlers); err != nil {
		return nil, fmt.Errorf("failed to configure club router: %w", err)
	}

	return &Module{
		ClubService:   service,
		ClubRouter:    clubRouter,
		observability: obs,
	}, nil
}

// Run starts the club module.
func (m *Module) Run(ctx context.Context, wg *sync.WaitGroup) {
	logger := m.observability.Provider.Logger
	logger.InfoContext(ctx, "Starting club module")

	ctx, cancel := context.WithCancel(ctx)
	m.cancelFunc = cancel
	defer cancel()

	if wg != nil {
		defer wg.Done()
	}

	<-ctx.Done()
	logger.InfoContext(ctx, "Club module goroutine stopped")
}

// Close shuts down the club module.
func (m *Module) Close() error {
	logger := m.observability.Provider.Logger
	logger.Info("Stopping club module")

	if m.cancelFunc != nil {
		m.cancelFunc()
	}

	if m.ClubRouter != nil {
		if err := m.ClubRouter.Close(); err != nil {
			logger.Error("Error closing ClubRouter from module", "error", err)
			return fmt.Errorf("error closing ClubRouter: %w", err)
		}
	}

	logger.Info("Club module stopped")
	return nil
}
