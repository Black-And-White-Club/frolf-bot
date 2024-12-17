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

	logger := watermill.NewStdLogger(false, false)

	router, pubSuber, err := watermillutil.NewRouter(natsURL, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Watermill router: %w", err)
	}

	commandBus, err := watermillutil.NewCommandBus(natsURL, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create command bus: %w", err)
	}

	modules, err := modules.NewModuleRegistry(dbService, commandBus, pubSuber)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize modules: %w", err)
	}

	// Register module handlers
	if err := RegisterHandlers(router, natsURL, logger, modules.UserModule, modules.RoundModule, modules.LeaderboardModule, modules.ScoreModule); err != nil {
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
