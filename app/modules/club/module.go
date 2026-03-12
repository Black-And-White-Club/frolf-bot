package club

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	clubservice "github.com/Black-And-White-Club/frolf-bot/app/modules/club/application"
	clubhandlers "github.com/Black-And-White-Club/frolf-bot/app/modules/club/infrastructure/handlers"
	clubqueue "github.com/Black-And-White-Club/frolf-bot/app/modules/club/infrastructure/queue"
	clubdb "github.com/Black-And-White-Club/frolf-bot/app/modules/club/infrastructure/repositories"
	clubrouter "github.com/Black-And-White-Club/frolf-bot/app/modules/club/infrastructure/router"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-chi/chi/v5"
	"github.com/uptrace/bun"
)

// Module represents the club module.
type Module struct {
	ClubService   clubservice.Service
	QueueService  clubqueue.QueueService
	ClubRouter    *clubrouter.ClubRouter
	cancelFunc    context.CancelFunc
	observability observability.Observability
}

// ClubModuleOptions groups all dependencies required to initialise the club module.
type ClubModuleOptions struct {
	Observability     observability.Observability
	EventBus          eventbus.EventBus
	Router            *message.Router
	Helpers           utils.Helpers
	RouterCtx         context.Context
	DB                *bun.DB
	HTTPRouter        chi.Router
	UserRepo          userdb.Repository
	RoundReader       roundservice.Service
	LeaderboardReader leaderboardservice.Service
	PostgresDSN       string
}

// NewClubModule creates and initializes a new club module.
func NewClubModule(ctx context.Context, opts ClubModuleOptions) (*Module, error) {
	obs := opts.Observability
	eventBus := opts.EventBus
	router := opts.Router
	helpers := opts.Helpers
	routerCtx := opts.RouterCtx
	db := opts.DB
	httpRouter := opts.HTTPRouter
	userRepo := opts.UserRepo
	roundReader := opts.RoundReader
	leaderboardReader := opts.LeaderboardReader
	postgresDSN := opts.PostgresDSN
	logger := obs.Provider.Logger
	tracer := obs.Registry.Tracer

	logger.InfoContext(ctx, "club.NewClubModule initializing")

	// 1. Initialize Repository
	repo := clubdb.NewRepository(db)

	// 2. Initialize Metrics
	metrics := obs.Registry.ClubMetrics

	queueService, err := clubqueue.NewService(ctx, db, logger, postgresDSN, metrics, eventBus, helpers)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize club queue service: %w", err)
	}
	if err := queueService.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start club queue service: %w", err)
	}

	// 3. Initialize Service (now includes userRepo for cross-module queries)
	service := clubservice.NewClubService(repo, userRepo, queueService, leaderboardReader, roundReader, logger, metrics, tracer, db)

	// 4. Initialize NATS event Handlers
	handlers := clubhandlers.NewClubHandlers(service, logger, tracer)

	// 5. Initialize NATS Router
	clubRouter := clubrouter.NewClubRouter(
		logger,
		router,
		eventBus,
		eventBus,
		helpers,
		tracer,
		metrics,
	)

	// 6. Configure the NATS router with handlers
	if err := clubRouter.Configure(routerCtx, handlers); err != nil {
		return nil, fmt.Errorf("failed to configure club router: %w", err)
	}

	// 7. Register HTTP routes for club discovery and invite management
	if httpRouter != nil {
		httpHandlers := clubhandlers.NewHTTPHandlers(service, userRepo, logger, tracer)
		httpRouter.Route("/api/clubs", func(r chi.Router) {
			// Public endpoints (no auth required)
			r.Get("/preview", httpHandlers.HandleGetInvitePreview)

			// Protected endpoints (auth middleware applied per-handler via cookie lookup)
			r.Get("/suggestions", httpHandlers.HandleGetSuggestions)
			r.Post("/join", httpHandlers.HandleJoinClub)
			r.Post("/join-by-code", httpHandlers.HandleJoinByCode)
			r.Route("/{uuid}", func(r chi.Router) {
				r.Get("/invites", httpHandlers.HandleListInvites)
				r.Post("/invites", httpHandlers.HandleCreateInvite)
				r.Delete("/invites/{code}", httpHandlers.HandleRevokeInvite)
			})
		})
	}

	return &Module{
		ClubService:   service,
		QueueService:  queueService,
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

	if err := closeModuleResources(logger, m.ClubRouter, m.QueueService); err != nil {
		return err
	}

	logger.Info("Club module stopped")
	return nil
}

type moduleCloser interface {
	Close() error
}

func closeModuleResources(logger *slog.Logger, router moduleCloser, queueService clubqueue.QueueService) error {
	var closeErr error

	if router != nil {
		if err := router.Close(); err != nil {
			logger.Error("Error closing ClubRouter from module", "error", err)
			closeErr = fmt.Errorf("error closing ClubRouter: %w", err)
		}
	}

	if queueService != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := queueService.Stop(ctx); err != nil {
			logger.Error("Error stopping club queue service", "error", err)
		}
	}

	return closeErr
}
