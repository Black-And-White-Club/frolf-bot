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
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-chi/chi/v5"
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
	httpRouter chi.Router,
	userRepo userdb.Repository,
) (*Module, error) {
	logger := obs.Provider.Logger
	tracer := obs.Registry.Tracer

	logger.InfoContext(ctx, "club.NewClubModule initializing")

	// 1. Initialize Repository
	repo := clubdb.NewRepository(db)

	// 2. Initialize Metrics (noop for now, real metrics wired via observability registry)
	metrics := clubmetrics.NewNoop()

	// 3. Initialize Service (now includes userRepo for cross-module queries)
	service := clubservice.NewClubService(repo, userRepo, logger, metrics, tracer, db)

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
