package betting

import (
	"context"
	"fmt"
	"sync"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	bettingservice "github.com/Black-And-White-Club/frolf-bot/app/modules/betting/application"
	bettinghandlers "github.com/Black-And-White-Club/frolf-bot/app/modules/betting/infrastructure/handlers"
	bettingdb "github.com/Black-And-White-Club/frolf-bot/app/modules/betting/infrastructure/repositories"
	bettingrouter "github.com/Black-And-White-Club/frolf-bot/app/modules/betting/infrastructure/router"
	bettingworkers "github.com/Black-And-White-Club/frolf-bot/app/modules/betting/infrastructure/workers"
	guilddb "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/infrastructure/repositories"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-chi/chi/v5"
	"github.com/uptrace/bun"
)

type Module struct {
	BettingService bettingservice.Service
	Router         *bettingrouter.Router
	marketWorker   *bettingworkers.MarketWorker
	cancelFunc     context.CancelFunc
	observability  observability.Observability
}

type ModuleOptions struct {
	Observability   observability.Observability
	EventBus        eventbus.EventBus
	Router          *message.Router
	Helpers         utils.Helpers
	RouterCtx       context.Context
	DB              *bun.DB
	HTTPRouter      chi.Router
	UserRepo        userdb.Repository
	GuildRepo       guilddb.Repository
	LeaderboardRepo leaderboarddb.Repository
	RoundRepo       rounddb.Repository
}

func NewModule(ctx context.Context, opts ModuleOptions) (*Module, error) {
	logger := opts.Observability.Provider.Logger
	tracer := opts.Observability.Registry.Tracer

	repo := bettingdb.NewRepository(opts.DB)
	service := bettingservice.NewService(repo, opts.UserRepo, opts.GuildRepo, opts.LeaderboardRepo, opts.RoundRepo, opts.Observability.Registry.BettingMetrics, logger, tracer, opts.DB)

	var lifecycleRouter *bettingrouter.Router
	if opts.Router != nil && opts.EventBus != nil {
		eventHandlers := bettinghandlers.NewEventHandlers(service, opts.Observability.Registry.BettingMetrics)
		lifecycleRouter = bettingrouter.NewRouter(logger, opts.Router, opts.EventBus, opts.EventBus, opts.Helpers, tracer)
		if err := lifecycleRouter.Configure(opts.RouterCtx, eventHandlers); err != nil {
			return nil, fmt.Errorf("failed to configure betting router: %w", err)
		}
	}

	if opts.HTTPRouter != nil {
		httpHandlers := bettinghandlers.NewHTTPHandlers(service, opts.UserRepo, logger, tracer, opts.Observability.Registry.BettingMetrics)
		opts.HTTPRouter.Route("/api/betting", func(r chi.Router) {
			r.Get("/overview", httpHandlers.HandleGetOverview)
			r.Get("/next-market", httpHandlers.HandleGetNextRoundMarket)
			r.Get("/admin/markets", httpHandlers.HandleGetAdminMarkets)
			r.Patch("/settings", httpHandlers.HandleUpdateSettings)
			r.Post("/bets", httpHandlers.HandlePlaceBet)
			r.Post("/admin/wallet-adjustments", httpHandlers.HandleAdjustWallet)
			r.Post("/admin/market-actions", httpHandlers.HandleAdminMarketAction)
		})
	}

	var marketWorker *bettingworkers.MarketWorker
	if opts.EventBus != nil && opts.RoundRepo != nil {
		discoverer := bettingworkers.NewRoundRepoGuildDiscoverer(opts.RoundRepo)
		marketWorker = bettingworkers.NewMarketWorker(service, discoverer, opts.EventBus, opts.Helpers, logger)
	}

	return &Module{
		BettingService: service,
		Router:         lifecycleRouter,
		marketWorker:   marketWorker,
		observability:  opts.Observability,
	}, nil
}

func (m *Module) Run(ctx context.Context, wg *sync.WaitGroup) {
	ctx, cancel := context.WithCancel(ctx)
	m.cancelFunc = cancel
	defer cancel()

	if wg != nil {
		defer wg.Done()
	}

	if m.marketWorker != nil {
		go m.marketWorker.Start(ctx)
	}

	<-ctx.Done()
}

func (m *Module) Close() error {
	if m.marketWorker != nil {
		m.marketWorker.Stop()
	}

	if m.cancelFunc != nil {
		m.cancelFunc()
	}

	if m.Router != nil {
		return m.Router.Close()
	}

	return nil
}
