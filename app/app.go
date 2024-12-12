package app

import (
	"context"
	"fmt"

	"github.com/Black-And-White-Club/tcr-bot/app/modules/user"
	usercommands "github.com/Black-And-White-Club/tcr-bot/app/modules/user/commands"
	userhandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/user/handlers"
	userqueries "github.com/Black-And-White-Club/tcr-bot/app/modules/user/queries"
	"github.com/Black-And-White-Club/tcr-bot/config"
	"github.com/Black-And-White-Club/tcr-bot/db/bundb"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
)

// App struct
type App struct {
	Cfg                *config.Config
	db                 *bundb.DBService
	UserCommandService usercommands.CommandService
	UserQueryService   userqueries.QueryService
	WatermillRouter    *message.Router
	WatermillPubSub    watermillutil.PubSuber
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

	// Initialize the NATS publisher
	// publisher, err := watermillutil.NewPublisher(natsURL, watermill.NewStdLogger(false, false))
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to create NATS publisher: %w", err)
	// }

	// Initialize the command bus
	commandBus, err := watermillutil.NewCommandBus(natsURL, watermill.NewStdLogger(false, false))
	if err != nil {
		return nil, fmt.Errorf("failed to create command bus: %w", err)
	}

	// Initialize user module
	userModule, err := initUserModule(dbService, commandBus, pubsub)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize user module: %w", err)
	}

	// Register module handlers
	if err := RegisterHandlers(router, pubsub, userModule); err != nil {
		return nil, fmt.Errorf("failed to register handlers: %w", err)
	}

	return &App{
		Cfg:                cfg,
		db:                 dbService,
		UserCommandService: userModule.CommandService, // Access from the module
		UserQueryService:   userModule.QueryService,   // Access from the module
		WatermillRouter:    router,
		WatermillPubSub:    pubsub,
	}, nil
}

func initUserModule(dbService *bundb.DBService, commandBus *cqrs.CommandBus, pubsub watermillutil.PubSuber) (*user.UserModule, error) { // Use *cqrs.CommandBus
	// Initialize UserHandlers
	userCommandService := usercommands.NewUserCommandService(
		dbService.User,
		pubsub,
		*commandBus,
	)
	userQueryService := userqueries.NewUserQueryService(
		dbService.User,
		pubsub.(*watermillutil.PubSub), // Add type assertion
	)

	// Register the user command handlers
	if err := userhandlers.RegisterUserCommandHandlers(commandBus, dbService.User, pubsub.(*watermillutil.PubSub)); err != nil {
		return nil, fmt.Errorf("failed to register user command handlers: %w", err)
	}

	// Initialize user module
	return user.NewUserModule(userCommandService, userQueryService, pubsub), nil
}

// DB returns the database service.
func (app *App) DB() *bundb.DBService {
	return app.db
}
