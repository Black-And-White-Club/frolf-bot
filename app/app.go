package app

import (
	"context"
	"fmt"
	"log"

	"github.com/Black-And-White-Club/tcr-bot/api/services"
	"github.com/Black-And-White-Club/tcr-bot/config"
	"github.com/Black-And-White-Club/tcr-bot/db/bundb"
	events "github.com/Black-And-White-Club/tcr-bot/event_bus"
	"github.com/Black-And-White-Club/tcr-bot/nats"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

type App struct {
	Cfg                *config.Config
	db                 *bundb.DBService
	NatsConnectionPool *nats.NatsConnectionPool
	LeaderboardService *services.LeaderboardService
	UserService        *services.UserService
	RoundService       *services.RoundService
	ScoreService       *services.ScoreService
	messagePublisher   message.Publisher
}

// NewApp initializes the application with the necessary services and configuration.
func NewApp(ctx context.Context) (*App, error) {
	cfg := config.NewConfig(ctx)
	dsn := cfg.DSN
	natsURL := cfg.NATS.URL

	// Initialize the database service
	dbService, err := bundb.NewBunDBService(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database service: %w", err)
	}

	// Initialize NATS connection pool
	natsConnectionPool, err := nats.NewNatsConnectionPool(natsURL, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize NATS connection pool: %w", err)
	}

	log.Printf("NATS connection pool initialized with URL: %s", natsURL)

	// Create the publisher
	publisher, err := events.NewPublisher(natsURL, watermill.NewStdLogger(false, false))
	if err != nil {
		return nil, fmt.Errorf("failed to create NATS publisher: %w", err)
	}

	// Initialize services with the correct types (pass publisher to services)
	leaderboardService := services.NewLeaderboardService(dbService.Leaderboard, natsConnectionPool, publisher)
	userService := services.NewUserService(dbService.User, natsConnectionPool, publisher)
	roundService := services.NewRoundService(dbService.Round, natsConnectionPool, publisher)
	scoreService := services.NewScoreService(dbService.Score, natsConnectionPool, publisher)

	return &App{
		Cfg:                cfg,
		db:                 dbService,
		NatsConnectionPool: natsConnectionPool,
		LeaderboardService: leaderboardService,
		UserService:        userService,
		RoundService:       roundService,
		ScoreService:       scoreService,
		messagePublisher:   publisher,
	}, nil
}

// DB returns the database service.
func (app *App) DB() *bundb.DBService {
	return app.db
}

// Publisher returns the publisher.
func (app *App) Publisher() message.Publisher {
	return app.messagePublisher
}
