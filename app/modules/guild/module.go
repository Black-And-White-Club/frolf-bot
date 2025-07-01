package guild

import (
	"context"
	"fmt"
	"sync"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	guildservice "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/application"
	guilddb "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/infrastructure/repositories"
	guildrouter "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/infrastructure/router"
	"github.com/Black-And-White-Club/frolf-bot/config"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/prometheus/client_golang/prometheus"
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

// NewGuildModule creates a new instance of the Guild module.
func NewGuildModule(
	ctx context.Context,
	cfg *config.Config,
	obs observability.Observability,
	guildDB guilddb.GuildDB,
	eventBus eventbus.EventBus,
	router *message.Router,
	helpers utils.Helpers,
	routerCtx context.Context,
) (*Module, error) {
	logger := obs.Provider.Logger
	metrics := obs.Registry.GuildMetrics
	tracer := obs.Registry.Tracer

	logger.InfoContext(ctx, "guild.NewGuildModule called")

	// Initialize guild service
	guildService := guildservice.NewGuildService(guildDB, eventBus, logger, metrics, tracer)

	// Create a Prometheus registry for this module (not used by router constructor)
	prometheusRegistry := prometheus.NewRegistry()

	// Initialize guild router (without prometheusRegistry)
	guildRouter := guildrouter.NewGuildRouter(logger, router, eventBus, eventBus, cfg, helpers, tracer)

	// Configure the router with the guild service
	if err := guildRouter.Configure(routerCtx, guildService, eventBus, metrics); err != nil {
		return nil, fmt.Errorf("failed to configure guild router: %w", err)
	}

	module := &Module{
		EventBus:           eventBus,
		GuildService:       guildService,
		config:             cfg,
		GuildRouter:        guildRouter,
		Helper:             helpers,
		observability:      obs,
		prometheusRegistry: prometheusRegistry,
	}

	return module, nil
}

// Run starts the guild module.
func (m *Module) Run(ctx context.Context, wg *sync.WaitGroup) {
	logger := m.observability.Provider.Logger
	logger.InfoContext(ctx, "Starting guild module")

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
	logger.InfoContext(ctx, "Guild module goroutine stopped")
}

// Close stops the guild module and cleans up resources.
func (m *Module) Close() error {
	logger := m.observability.Provider.Logger
	logger.Info("Stopping guild module")

	// Cancel any other running operations
	if m.cancelFunc != nil {
		m.cancelFunc()
	}

	// Close the GuildRouter
	if m.GuildRouter != nil {
		if err := m.GuildRouter.Close(); err != nil {
			logger.Error("Error closing GuildRouter from module", "error", err)
			return fmt.Errorf("error closing GuildRouter: %w", err)
		}
	}

	logger.Info("Guild module stopped")
	return nil
}
