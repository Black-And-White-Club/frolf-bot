package app

import (
	"context"
	"fmt"
	"log"

	"github.com/Black-And-White-Club/tcr-bot/config"
	"github.com/Black-And-White-Club/tcr-bot/db/bundb"
	natsjetstream "github.com/Black-And-White-Club/tcr-bot/nats"
	"github.com/Black-And-White-Club/tcr-bot/round"
	roundcommands "github.com/Black-And-White-Club/tcr-bot/round/commands"
	roundapi "github.com/Black-And-White-Club/tcr-bot/round/commandsinterface"
	roundconverter "github.com/Black-And-White-Club/tcr-bot/round/converter"
	rounddb "github.com/Black-And-White-Club/tcr-bot/round/db"
	roundevents "github.com/Black-And-White-Club/tcr-bot/round/eventhandling"
	roundqueries "github.com/Black-And-White-Club/tcr-bot/round/queries"
	usercommands "github.com/Black-And-White-Club/tcr-bot/user/commands"
	userdb "github.com/Black-And-White-Club/tcr-bot/user/db"
	userqueries "github.com/Black-And-White-Club/tcr-bot/user/queries"
	"github.com/Black-And-White-Club/tcr-bot/watermillcmd"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type App struct {
	Cfg                *config.Config
	db                 *bundb.DBService
	NatsConnectionPool *natsjetstream.NatsConnectionPool
	// LeaderboardService *leaderboard.LeaderboardService
	UserService       usercommands.UserService
	UserQueryService  userqueries.UserQueryService
	RoundService      roundapi.CommandService
	RoundQueryService roundqueries.QueryService
	// ScoreService       *score.ScoreService
	messagePublisher  message.Publisher
	roundDB           rounddb.RoundDB
	userDB            userdb.UserDB
	roundEventHandler round.RoundEventHandler
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

	natsConnectionPool, err := natsjetstream.NewNatsConnectionPool(natsURL, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize NATS connection pool: %w", err)
	}

	log.Printf("NATS connection pool initialized with URL: %s", natsURL)

	publisher, err := natsjetstream.NewPublisher(natsURL, watermill.NewStdLogger(false, false))
	if err != nil {
		return nil, fmt.Errorf("failed to create NATS publisher: %w", err)
	}

	// Initialize the command bus
	commandBus, err := watermillcmd.NewCommandBus(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create command bus: %w", err)
	}

	// Initialize UserHandlers
	userService := usercommands.NewUserCommandService(
		dbService.User,
		app.NatsConnectionPool,
		publisher,
		commandBus,
	)
	userQueryService := userqueries.NewUserQueryService(
		dbService.User,
		watermillcmd.NewNatsPubSub(),
	)

	// Register the user command handlers
	watermillcmd.RegisterUserCommandHandlers(commandBus, dbService.User, watermillcmd.NewNatsPubSub())

	natsjetstream.InitPublisher(publisher)

	// Initialize roundService with a nil roundEventHandler initially
	roundService := roundcommands.NewRoundCommandService(dbService.Round, &roundconverter.DefaultRoundConverter{}, publisher, nil) // Inject the converter

	// Initialize roundEventHandler
	roundEventHandler := roundevents.NewRoundEventHandler(roundService, publisher)

	// Assign the roundEventHandler to the roundService
	roundService.SetEventHandler(roundEventHandler)

	roundQueryService := roundqueries.NewRoundQueryService(dbService.Round, &roundconverter.DefaultRoundConverter{}) // Inject the converter

	return &App{
		Cfg:                cfg,
		db:                 dbService,
		NatsConnectionPool: natsConnectionPool,
		// LeaderboardService: leaderboardService,
		userDB:            dbService.User,
		UserService:       userService,
		UserQueryService:  userQueryService,
		RoundService:      roundService,
		RoundQueryService: roundQueryService,
		// ScoreService:       scoreService,
		messagePublisher:  publisher,
		roundDB:           dbService.Round,
		roundEventHandler: roundEventHandler,
	}, nil
}

func (app *App) applyMiddleware(r chi.Router) chi.Router {
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	// Add any other middleware you need here, like authentication, authorization, etc.

	return r
}

// DB returns the database service.
func (app *App) DB() *bundb.DBService {
	return app.db
}

// Publisher returns the publisher.
func (app *App) Publisher() message.Publisher {
	return app.messagePublisher
}
