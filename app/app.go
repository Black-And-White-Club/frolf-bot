package app

import (
	"context"
	"fmt"

	"github.com/Black-And-White-Club/tcr-bot/config"
	"github.com/Black-And-White-Club/tcr-bot/db/bundb"
	"github.com/Black-And-White-Club/tcr-bot/internal/modules"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// App struct
type App struct {
	Cfg             *config.Config
	db              *bundb.DBService
	WatermillRouter *message.Router
	Modules         *modules.ModuleRegistry
}

// NewApp initializes the application with the necessary services and configuration.
func NewApp(ctx context.Context) (*App, error) {
	cfg := config.NewConfig(ctx)
	dsn := cfg.DSN
	natsURL := cfg.NATS.URL

	dbService, err := bundb.NewBunDBService(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database service: %w", err)
	}

	// Initialize the Watermill router and pubsub
	router, pubsub, err := watermillutil.NewRouter(natsURL, watermill.NewStdLogger(false, false))
	if err != nil {
		return nil, fmt.Errorf("failed to create Watermill router: %w", err)
	}

	// Initialize the command bus
	commandBus, err := watermillutil.NewCommandBus(natsURL, watermill.NewStdLogger(false, false))
	if err != nil {
		return nil, fmt.Errorf("failed to create command bus: %w", err)
	}

	// Initialize module registry
	modules, err := modules.NewModuleRegistry(dbService, commandBus, pubsub) // Add commandBus argument
	if err != nil {
		return nil, fmt.Errorf("failed to initialize modules: %w", err)
	}

	// Register module handlers
	if err := RegisterHandlers(router, pubsub, modules.UserModule); err != nil {
		return nil, fmt.Errorf("failed to register handlers: %w", err)
	}

	return &App{
		Cfg:             cfg,
		db:              dbService,
		WatermillRouter: router,
		Modules:         modules,
	}, nil
}

// DB returns the database service.
func (app *App) DB() *bundb.DBService {
	return app.db
}
