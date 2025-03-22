package leaderboard

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	leaderboardrouter "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/router"
	"github.com/Black-And-White-Club/frolf-bot/config"
	"github.com/ThreeDotsLabs/watermill/message"
)

type Module struct {
	EventBus           eventbus.EventBus
	LeaderboardService leaderboardservice.Service
	logger             *slog.Logger
	config             *config.Config
	LeaderboardRouter  *leaderboardrouter.LeaderboardRouter
	cancelFunc         context.CancelFunc
	helper             utils.Helpers
}

func NewLeaderboardModule(ctx context.Context, cfg *config.Config, logger *slog.Logger, leaderboardDB leaderboarddb.LeaderboardDB, eventBus eventbus.EventBus, router *message.Router, helper utils.Helpers) (*Module, error) {

	leaderboardService := leaderboardservice.NewLeaderboardService(leaderboardDB, eventBus, logger)
	leaderboardRouter := leaderboardrouter.NewLeaderboardRouter(logger, router, eventBus, helper)

	if err := leaderboardRouter.Configure(leaderboardService); err != nil {
		logger.Error("❌ Failed to configure leaderboard router", slog.Any("error", err))
		return nil, fmt.Errorf("failed to configure leaderboard router: %w", err)
	}

	module := &Module{
		EventBus:           eventBus,
		LeaderboardService: leaderboardService,
		logger:             logger,
		config:             cfg,
		LeaderboardRouter:  leaderboardRouter,
		helper:             helper,
	}

	return module, nil
}

func (m *Module) Run(ctx context.Context, wg *sync.WaitGroup) {
	m.logger.Info("Starting leaderboard module")

	// Create a context that can be canceled
	ctx, cancel := context.WithCancel(ctx)
	m.cancelFunc = cancel
	defer cancel()

	// Keep this goroutine alive until the context is canceled
	<-ctx.Done()
	m.logger.Info("Leaderboard module goroutine stopped")
}

func (m *Module) Close() error {
	m.logger.Info("Stopping leaderboard module")

	// Cancel any other running operations
	if m.cancelFunc != nil {
		m.cancelFunc()
	}

	m.logger.Info("Leaderboard module stopped")
	return nil
}
