package leaderboard

import (
	"context"
	"fmt"
	"sync"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	leaderboardhandlers "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/handlers"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	leaderboardrouter "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/router"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/saga"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
	"github.com/Black-And-White-Club/frolf-bot/config"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/uptrace/bun"
)

type Module struct {
	EventBus           eventbus.EventBus
	LeaderboardService leaderboardservice.Service
	SagaCoordinator    *saga.SwapSagaCoordinator
	config             *config.Config
	LeaderboardRouter  *leaderboardrouter.LeaderboardRouter
	cancelFunc         context.CancelFunc
	Helper             utils.Helpers
	observability      observability.Observability
	prometheusRegistry *prometheus.Registry
}

func NewLeaderboardModule(
	ctx context.Context,
	cfg *config.Config,
	obs observability.Observability,
	db *bun.DB,
	leaderboardDB leaderboarddb.Repository,
	eventBus eventbus.EventBus,
	router *message.Router,
	helpers utils.Helpers,
	routerCtx context.Context,
	js jetstream.JetStream,
	userService userservice.Service,
) (*Module, error) {
	logger := obs.Provider.Logger
	metrics := obs.Registry.LeaderboardMetrics
	tracer := obs.Registry.Tracer

	logger.InfoContext(ctx, "Initializing Leaderboard Module")

	// 1. Domain Service (The Business Logic)
	service := leaderboardservice.NewLeaderboardService(db, leaderboardDB, logger, metrics, tracer)

	// 2. Saga Infrastructure (The "Memory" for swaps)
	kv, err := ensureSagaKV(js)
	if err != nil {
		return nil, err
	}
	sagaCoord := saga.NewSwapSagaCoordinator(kv, service, logger)

	// 3. Communications (The Router/Traffic Cop)
	promRegistry := prometheus.NewRegistry()
	lbRouter := leaderboardrouter.NewLeaderboardRouter(logger, router, eventBus, eventBus, cfg, helpers, tracer, promRegistry)

	handlers := leaderboardhandlers.NewLeaderboardHandlers(service, userService, sagaCoord, logger, tracer, helpers, metrics)

	if err := lbRouter.Configure(routerCtx, handlers); err != nil {
		return nil, fmt.Errorf("failed to configure leaderboard router: %w", err)
	}

	return &Module{
		EventBus:           eventBus,
		LeaderboardService: service,
		SagaCoordinator:    sagaCoord,
		config:             cfg,
		LeaderboardRouter:  lbRouter,
		Helper:             helpers,
		observability:      obs,
		prometheusRegistry: promRegistry,
	}, nil
}

// ensureSagaKV is a private helper to keep the constructor clean.
func ensureSagaKV(js jetstream.JetStream) (jetstream.KeyValue, error) {
	kv, err := js.KeyValue(context.Background(), "tag_swap_intents")
	if err == nil {
		return kv, nil
	}

	kv, err = js.CreateKeyValue(context.Background(), jetstream.KeyValueConfig{
		Bucket: "tag_swap_intents",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to bind saga KV bucket: %w", err)
	}

	return kv, nil
}

func (m *Module) Run(ctx context.Context, wg *sync.WaitGroup) {
	m.observability.Provider.Logger.InfoContext(ctx, "Starting leaderboard module")
	ctx, cancel := context.WithCancel(ctx)
	m.cancelFunc = cancel

	if wg != nil {
		wg.Add(1)
		defer wg.Done()
	}

	<-ctx.Done()
	m.observability.Provider.Logger.InfoContext(ctx, "Leaderboard module goroutine stopped")
}

func (m *Module) Close() error {
	if m.cancelFunc != nil {
		m.cancelFunc()
	}
	if m.LeaderboardRouter != nil {
		return m.LeaderboardRouter.Close()
	}
	return nil
}
