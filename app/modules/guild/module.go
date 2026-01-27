package guild

import (
	"context"
	"fmt"
	"sync"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	guildservice "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/application"
	guildhandlers "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/infrastructure/handlers"
	guilddb "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/infrastructure/repositories"
	guildrouter "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/infrastructure/router"
	"github.com/Black-And-White-Club/frolf-bot/config"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/uptrace/bun"
)

// Module represents the guild module.
type Module struct {
	EventBus           eventbus.EventBus
	GuildService       guildservice.Service
	config             *config.Config
	GuildRouter        *guildrouter.GuildRouter
	cancelFunc         context.CancelFunc
	Helper             utils.Helpers
	observability      observability.Observability
	prometheusRegistry *prometheus.Registry
}

// NewGuildModule creates and initializes a new guild module.
func NewGuildModule(
	ctx context.Context,
	cfg *config.Config,
	obs observability.Observability,
	guildRepo guilddb.Repository,
	eventBus eventbus.EventBus,
	router *message.Router,
	helpers utils.Helpers,
	routerCtx context.Context,
	db *bun.DB,
) (*Module, error) {
	logger := obs.Provider.Logger
	metrics := obs.Registry.GuildMetrics
	tracer := obs.Registry.Tracer

	logger.InfoContext(ctx, "guild.NewGuildModule initializing")

	// 1. Initialize Service
	service := guildservice.NewGuildService(guildRepo, logger, metrics, tracer, db)

	// 2. Initialize Handlers
	handlers := guildhandlers.NewGuildHandlers(service, logger, tracer, helpers, metrics)

	// 3. Initialize Router
	prometheusRegistry := prometheus.NewRegistry()
	guildRouter := guildrouter.NewGuildRouter(
		logger,
		router,
		eventBus,
		eventBus,
		cfg,
		helpers,
		tracer,
		prometheusRegistry,
	)

	// 4. Configure the router with handlers
	if err := guildRouter.Configure(routerCtx, handlers); err != nil {
		return nil, fmt.Errorf("failed to configure guild router: %w", err)
	}

	return &Module{
		EventBus:           eventBus,
		GuildService:       service,
		config:             cfg,
		GuildRouter:        guildRouter,
		Helper:             helpers,
		observability:      obs,
		prometheusRegistry: prometheusRegistry,
	}, nil
}

// Run starts the guild module.
func (m *Module) Run(ctx context.Context, wg *sync.WaitGroup) {
	m.observability.Provider.Logger.InfoContext(ctx, "Starting guild module")
	ctx, cancel := context.WithCancel(ctx)
	m.cancelFunc = cancel

	if wg != nil {
		wg.Add(1)
		defer wg.Done()
	}

	<-ctx.Done()
	m.observability.Provider.Logger.InfoContext(ctx, "Guild module goroutine stopped")
}

// Close shuts down the guild module.
func (m *Module) Close() error {
	if m.cancelFunc != nil {
		m.cancelFunc()
	}
	if m.GuildRouter != nil {
		return m.GuildRouter.Close()
	}
	return nil
}
